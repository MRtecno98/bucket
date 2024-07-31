package repositories

// Use this library to implement a repository.
import (
	"context"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/MRtecno98/bucket/bucket"
	"github.com/MRtecno98/bucket/bucket/platforms"
	"github.com/MRtecno98/bucket/bucket/repositories/spigotmc"
	"github.com/sunxyw/go-spiget/spiget"
	"golang.org/x/exp/slices"
)

// TODO: SpigotMC repository format (https://spiget.org/)

const SPIGOTMC_REPOSITORY = "spigotmc"

type SpigotMC struct {
	bucket.LockRepository

	Client *spiget.Client

	categoryNames map[int]string
}

type SpigotResource struct {
	spiget.Resource

	repository *SpigotMC
	completed  bool
}

type SpigotVersionInfo struct {
	*SpigotResource
	spiget.Version
}

type SpigotVersion struct {
	SpigotVersionInfo

	Name        string        `json:"name"`
	ReleaseDate int64         `json:"releaseDate"`
	Downloads   int           `json:"downloads"`
	Rating      spiget.Rating `json:"rating"`
}

type SpigotFile struct {
	*SpigotVersion
}

func init() {
	bucket.RegisterRepository(SPIGOTMC_REPOSITORY,
		func(ctx context.Context, oc *bucket.OpenContext, opts map[string]string) bucket.Repository {
			return NewSpigotRepository(ctx, oc) // Go boilerplate
		})
}

func NewSpigotRepository(ctx context.Context, context *bucket.OpenContext) *SpigotMC {
	return &SpigotMC{
		LockRepository: bucket.LockRepository{Lock: ctx},
		Client:         spiget.NewClient(nil),
	}
}

func (r *SpigotMC) Provider() string {
	return SPIGOTMC_REPOSITORY
}

func (r *SpigotMC) Resolve(plugin bucket.Plugin) (bucket.RemotePlugin, []bucket.RemotePlugin, error) {
	var tot int
	var res []bucket.RemotePlugin

	for _, name := range bucket.Distinct([]string{
		plugin.GetName(), bucket.Decamel(plugin.GetName(), " ")}) {
		cand, n, err := r.Search(name, 5)
		if err != nil {
			return nil, nil, err
		}

		tot += n
		res = append(res, cand...)
	}

	if meta, ok := plugin.(bucket.PluginMetadata); ok {
		for _, a := range meta.GetAuthors() {
			auts, err := r.GetAuthor(strings.ReplaceAll(a, " ", ""))
			if err != nil {
				continue
			}

			for _, aut := range auts {
				autres, rsp, err := r.GetAuthorResources(aut)
				if rsp.StatusCode == 404 {
					continue
				} else if err != nil {
					return nil, nil, err
				}

				tot += len(autres)
				for _, ares := range autres {
					res = append(res, ares)
				}
			}
		}
	}

	if tot == 0 {
		return nil, nil, r.parseError(fmt.Errorf("no match found for \"%s\"", plugin.GetName()))
	}

	return res[0], res, nil
}

func (r *SpigotMC) Get(identifier string) (bucket.RemotePlugin, error) {
	i, err := strconv.Atoi(identifier)
	if err != nil {
		return nil, r.parseError(err)
	}

	res, _, err := r.Client.Resources.Get(r.Lock, i)

	return &SpigotResource{repository: r, Resource: *res}, r.parseError(err)
}

func (r *SpigotMC) SearchAll(query string, max int) ([]bucket.RemotePlugin, int, error) {
	return r.Search(query, max)
}

func (r *SpigotMC) Search(query string, max int) ([]bucket.RemotePlugin, int, error) {
	res, rsp, err := r.Client.Search.SearchResource(r.Lock, query,
		&spiget.ResourceSearchOptions{})

	if rsp.StatusCode == 404 {
		return []bucket.RemotePlugin{}, 0, nil
	} else if err != nil {
		return nil, 0, r.parseError(err)
	}

	plugins := make([]bucket.RemotePlugin, 0, len(res))
	for _, p := range res {
		plugins = append(plugins, &SpigotResource{repository: r, Resource: *p})
	}

	return plugins, len(plugins), nil
}

func (r *SpigotMC) InitCategoryNames() error {
	cats, _, err := r.Client.Categories.List(r.Lock,
		&spiget.CategoryListOptions{ListOptions: spiget.ListOptions{Size: 100}})

	if err != nil {
		return err
	}

	r.categoryNames = make(map[int]string, len(cats))
	for _, c := range cats {
		r.categoryNames[c.ID] = c.Name
	}

	return nil
}

func (r *SpigotMC) GetAuthor(name string) ([]*spiget.Author, error) {
	auths, rsp, err := r.Client.Authors.Search(r.Lock, name, &spiget.AuthorSearchOptions{})
	if rsp.StatusCode == 404 {
		return []*spiget.Author{}, nil
	}

	return auths, err
}

func (r *SpigotMC) GetAuthorResources(author *spiget.Author) ([]*SpigotResource, *spiget.Response, error) {
	url := fmt.Sprintf("authors/%d/resources", author.ID)
	req, err := r.Client.NewRequest("GET", url, nil)
	if err != nil {
		return nil, nil, r.parseError(err)
	}

	var resources []spiget.Resource
	rsp, err := r.Client.Do(r.Lock, req, &resources)
	if err != nil {
		return nil, rsp, r.parseError(err)
	}

	spigot := make([]*SpigotResource, 0, len(resources))
	for _, res := range resources {
		spigot = append(spigot, &SpigotResource{repository: r, Resource: res})
	}

	return spigot, rsp, nil
}

func (r *SpigotMC) CategoryNames() (map[int]string, error) {
	if r.categoryNames == nil {
		err := r.InitCategoryNames()
		if err != nil {
			return nil, r.parseError(err)
		}
	}

	return r.categoryNames, nil
}

func (r *SpigotResource) GetName() string {
	return r.Name
}

func (r *SpigotResource) requireComplete() error {
	if !r.completed {
		res, err := r.repository.Get(r.GetIdentifier())
		if err != nil {
			return err
		}

		*r = *res.(*SpigotResource)
		r.completed = true
	}

	return nil
}

func (r *SpigotResource) GetLatestVersion() (bucket.RemoteVersion, error) {
	vers, err := r.GetVersions()
	if err != nil {
		return nil, r.repository.parseError(err)
	}

	return vers[0], nil
}

func (r *SpigotResource) GetVersion(identifier string) (bucket.RemoteVersion, error) {
	i, err := strconv.Atoi(identifier)
	if err != nil {
		return nil, r.repository.parseError(err)
	}

	for _, v := range r.Versions {
		if v.ID == i {
			return (&SpigotVersionInfo{SpigotResource: r, Version: v}).Get()
		}
	}

	return nil, r.repository.parseError(fmt.Errorf("version %s not found", identifier))
}

func (r *SpigotResource) GetVersions() ([]bucket.RemoteVersion, error) {
	err := r.requireComplete()
	if err != nil {
		return nil, err
	}

	vers := make([]bucket.RemoteVersion, 0, len(r.Versions))
	for _, v := range r.Versions {
		vers = append(vers, &SpigotVersion{SpigotVersionInfo: SpigotVersionInfo{SpigotResource: r, Version: v}})
	}

	return vers, nil
}

func (r *SpigotResource) GetVersionIdentifiers() ([]string, error) {
	vers, err := r.GetVersions()
	if err != nil {
		return nil, r.repository.parseError(err)
	}

	var identifiers []string
	for _, v := range vers {
		identifiers = append(identifiers, v.GetIdentifier())
	}

	return identifiers, nil
}

func (r *SpigotResource) GetLatestCompatible(platform bucket.PlatformType) (bucket.RemoteVersion, error) {
	vers, err := r.GetVersions()
	if err != nil {
		return nil, r.repository.parseError(err)
	}

	for _, v := range vers {
		if v.Compatible(platform) {
			return v, nil
		}
	}

	return nil, r.repository.parseError(fmt.Errorf("no compatible version found"))
}

func (s *SpigotMC) categoryCompatible(scat spiget.Category, platform bucket.PlatformType) bool {
	cat, err := spigotmc.GetCategory(scat)
	if err != nil {
		return false
	}

	for _, p := range cat.CompatiblePlatforms {
		if p == platform.Name {
			return true
		}
	}

	return false
}

func (r *SpigotResource) Compatible(platform bucket.PlatformType) bool {
	res := r.repository.categoryCompatible(r.Category, platform)
	if !res {
		v, err := r.GetLatestCompatible(platform)
		return err == nil && v != nil
	}

	return res
}

func (r *SpigotResource) GetIdentifier() string {
	return strconv.Itoa(r.ID)
}

func (r *SpigotResource) GetAuthors() []string {
	var contrs []string
	if r.Contributors != "" {
		contrs = strings.Split(r.Contributors, ",")
	} else {
		contrs = []string{}
	}

	if r.Author.Name != "" {
		return append(contrs, r.Author.Name)
	} else {
		aut, _, err := r.repository.Client.Authors.Get(r.repository.Lock, r.Author.ID)
		if err != nil {
			return contrs
		} else {
			return append(contrs, aut.Name)
		}
	}

}

func (r *SpigotResource) GetDescription() string {
	return r.Tag
}

func (r *SpigotResource) GetWebsite() string {
	l, ok := r.Links["additionalInformation"]
	if ok {
		return l
	}

	return r.SourceCodeLink
}

func (v *SpigotVersionInfo) GetIdentifier() string {
	return strconv.Itoa(v.ID)
}

func (v *SpigotVersionInfo) GetName() string {
	return v.UUID
}

func (v *SpigotVersionInfo) GetCategory() (*spigotmc.Category, error) {
	return spigotmc.GetCategory(v.Category)
}

func (v *SpigotVersionInfo) Compatible(platform bucket.PlatformType) bool {
	return v.repository.categoryCompatible(v.Category, platform)
}

func (v *SpigotVersionInfo) Get() (*SpigotVersion, error) {
	u := fmt.Sprintf("resources/%d/versions/%d", v.Resource.ID, v.ID)
	req, err := v.repository.Client.NewRequest("GET", u, nil)
	if err != nil {
		return nil, v.repository.parseError(err)
	}

	ver := SpigotVersion{SpigotVersionInfo: *v}
	_, err = v.repository.Client.Do(v.repository.Lock, req, &ver)

	return &ver, v.repository.parseError(err)
}

func (v *SpigotVersion) GetName() string {
	return v.Name
}

func (v *SpigotVersion) GetDependencies() []bucket.Dependency {
	panic("not implemented") // TODO: Implement
}

func (v *SpigotVersion) Compatible(platform bucket.PlatformType) bool {
	return slices.Contains(platform.EveryCompatible(), platforms.SpigotTypePlatform.Name)
}

func (v *SpigotVersion) GetFiles() ([]bucket.RemoteFile, error) {
	return []bucket.RemoteFile{&SpigotFile{SpigotVersion: v}}, nil
}

func (f *SpigotFile) Name() string {
	return f.SpigotVersion.Name
}

func (f *SpigotFile) Optional() bool {
	return false
}

func (f *SpigotFile) Download() (io.ReadCloser, error) {
	if f.External {
		return nil, f.repository.parseError(fmt.Errorf("external file not supported"))
	}

	r, err := f.repository.Client.Resources.DownloadVersion(f.repository.Lock, f.Resource.ID, f.ID)
	if err != nil {
		return nil, f.repository.parseError(err)
	}

	return r.Body, nil
}

func (f *SpigotFile) Verify() error {
	return nil // SpigotMC does not provide checksums
}

func (m *SpigotMC) parseError(err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("spigotmc: %s", err)
}
