package repositories

// Use this library to implement a repository.
import (
	"context"
	"fmt"
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

func (r *SpigotMC) Resolve(plugin bucket.Plugin) (bucket.RemotePlugin, []bucket.RemotePlugin, error) {
	return nil, nil, fmt.Errorf("spigotmc: not implemented")
}

func (r *SpigotMC) Get(identifier string) (bucket.RemotePlugin, error) {
	i, err := strconv.Atoi(identifier)
	if err != nil {
		return nil, err
	}

	res, _, err := r.Client.Resources.Get(r.Lock, i)

	return &SpigotResource{repository: r, Resource: *res}, err
}

func (r *SpigotMC) GetByHash(hash string) (bucket.RemoteVersion, error) {
	return nil, nil
}

func (r *SpigotMC) SearchAll(query string, max int) ([]bucket.RemotePlugin, int, error) {
	return nil, 0, nil
}

func (r *SpigotMC) Search(query string, max int) ([]bucket.RemotePlugin, int, error) {
	return nil, 0, nil
}

func (r *SpigotMC) GetVersion(identifier string) (bucket.RemoteVersion, error) {
	return nil, nil
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

func (r *SpigotMC) CategoryNames() (map[int]string, error) {
	if r.categoryNames == nil {
		err := r.InitCategoryNames()
		if err != nil {
			return nil, err
		}
	}

	return r.categoryNames, nil
}

func (r *SpigotResource) GetName() string {
	return r.Name
}

func (r *SpigotResource) GetLatestVersion() (bucket.RemoteVersion, error) {
	vers, err := r.GetVersions()
	if err != nil {
		return nil, err
	}

	return vers[0], nil
}

func (r *SpigotResource) GetVersion(identifier string) (bucket.RemoteVersion, error) {
	i, err := strconv.Atoi(identifier)
	if err != nil {
		return nil, err
	}

	for _, v := range r.Versions {
		if v.ID == i {
			return &SpigotVersion{SpigotVersionInfo: SpigotVersionInfo{SpigotResource: r, Version: v}}, nil
		}
	}

	return nil, fmt.Errorf("version %s not found", identifier)
}

func (r *SpigotResource) GetVersions() ([]bucket.RemoteVersion, error) {
	vers := make([]bucket.RemoteVersion, 0, len(r.Versions))
	for _, v := range r.Versions {
		vers = append(vers, &SpigotVersion{SpigotVersionInfo: SpigotVersionInfo{SpigotResource: r, Version: v}})
	}

	return vers, nil
}

func (r *SpigotResource) GetVersionIdentifiers() ([]string, error) {
	vers, err := r.GetVersions()
	if err != nil {
		return nil, err
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
		return nil, err
	}

	for _, v := range vers {
		if v.Compatible(platform) {
			return v, nil
		}
	}

	return nil, parseError(fmt.Errorf("no compatible version found"))
}

func (r *SpigotResource) Compatible(platform bucket.PlatformType) bool {
	ver, err := r.GetLatestCompatible(platform)
	return err == nil && ver != nil
}

func (r *SpigotResource) GetIdentifier() string {
	return strconv.Itoa(r.ID)
}

func (r *SpigotResource) GetAuthors() []string {
	if r.Contributors != "" {
		return strings.Split(r.Contributors, ",")
	}

	return []string{r.Author.Name}
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
	cat, err := v.GetCategory()
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

func (v *SpigotVersionInfo) Get() (*SpigotVersion, error) {
	u := fmt.Sprintf("resources/%d/versions/%d", v.Resource.ID, v.ID)
	req, err := v.repository.Client.NewRequest("GET", u, nil)
	if err != nil {
		return nil, err
	}

	ver := SpigotVersion{SpigotVersionInfo: *v}
	_, err = v.repository.Client.Do(v.repository.Lock, req, &ver)

	return &ver, err
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
	var remoteFiles []bucket.RemoteFile
	/* for i := range v.File {
		remoteFiles = append(remoteFiles, &p.Files[i])
	} */

	return remoteFiles, nil
}
