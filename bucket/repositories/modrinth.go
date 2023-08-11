package repositories

import (
	"context"
	"crypto/sha1"
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"hash"
	"io"
	"sort"
	"strconv"

	"github.com/MRtecno98/bucket/bucket"
	"github.com/go-resty/resty/v2"
	"golang.org/x/exp/slices"
)

// TODO: Modrinth repository format (https://modrinth.com/api/docs)

const MODRINTH_ENDPOINT = "https://api.modrinth.com/v2"

const MODRINTH_REPOSITORY = "modrinth"

type ModrinthSide string
type ModrinthType string
type ModrinthDependency string
type ModrinthVersionType string
type ModrinthMonetization string

const (
	SIDE_REQUIRED    ModrinthSide = "required"
	SIDE_OPTIONAL    ModrinthSide = "optional"
	SIDE_UNSUPPORTED ModrinthSide = "unsupported"

	PROJECT_MOD          ModrinthType = "mod"
	PROJECT_MODPACK      ModrinthType = "modpack"
	PROJECT_RESOURCEPACK ModrinthType = "resourcepack"

	DEPENDENCY_REQUIRED ModrinthDependency = "required"
	DEPENDENCY_OPTIONAL ModrinthDependency = "optional"
	DEPENDENCY_INCOMPAT ModrinthDependency = "incompatible"
	DEPENDENCY_EMBEDDED ModrinthDependency = "embedded"

	MODRINTH_RELEASE ModrinthVersionType = "release"
	MODRINTH_BETA    ModrinthVersionType = "beta"
	MODRINTH_ALPHA   ModrinthVersionType = "alpha"

	MODRINTH_MONETIZED         ModrinthMonetization = "monetized"
	MODRINTH_DEMONETIZED       ModrinthMonetization = "demonetized"
	MODRINTH_FORCE_DEMONETIZED ModrinthMonetization = "force-demonetized"
)

type ModrinthProject struct {
	repository *Modrinth `json:"-"`

	ID            string       `json:"id"`
	Slug          string       `json:"slug"`
	Title         string       `json:"title"`
	Description   string       `json:"description"`
	Categories    []string     `json:"categories"`
	ClientSide    ModrinthSide `json:"client_side"`
	ServerSide    ModrinthSide `json:"server_side"`
	Body          string       `json:"body"`
	AdtCategories []string     `json:"additional_categories"`
	IssuesUrl     string       `json:"issues_url"`
	SourceUrl     string       `json:"source_url"`
	WikiUrl       string       `json:"wiki_url"`
	DiscordUrl    string       `json:"discord_url"`
	ProjectType   ModrinthType `json:"project_type"`
	Downloads     int          `json:"downloads"`
	IconUrl       string       `json:"icon_url"`
	Team          string       `json:"team"`
	Published     string       `json:"published"`
	Updated       string       `json:"updated"`
	Followers     int          `json:"followers"`
	Versions      []string     `json:"versions"`
	Gallery       []string     `json:"gallery"`

	License struct {
		ID   string `json:"id"`
		Name string `json:"name"`
		Url  string `json:"url"`
	} `json:"license"`

	DonationUrls []struct {
		ID       string `json:"id"`
		Platform string `json:"platform"`
		Url      string `json:"url"`
	} `json:"donation_urls"`
}

type ModrinthVersion struct {
	ModrinthProject `json:"-"`

	ID            string `json:"id"`
	Name          string `json:"name"`
	VersionNumber string `json:"version_number"`
	Changelog     string `json:"changelog"`
	Dependencies  []struct {
		VersionID string             `json:"version_id"`
		ProjectID string             `json:"project_id"`
		FileName  string             `json:"file_name"`
		Type      ModrinthDependency `json:"dependency_type"`
	} `json:"dependencies"`

	GameVersions []string            `json:"game_versions"`
	Type         ModrinthVersionType `json:"version_type"`
	Loaders      []string            `json:"loaders"`
	ProjectID    string              `json:"project_id"`
	AuthorID     string              `json:"author_id"`
	Published    string              `json:"date_published"`
	Downloads    int                 `json:"downloads"`

	Files []ModrinthFile `json:"files"`
}

type ModrinthFile struct {
	ModrinthVersion `json:"-"`

	hasher hash.Hash

	Hashes struct {
		Sha1   string `json:"sha1"`
		Sha512 string `json:"sha512"`
	} `json:"hashes"`

	URL      string `json:"url"`
	Filename string `json:"filename"`
	Primary  bool   `json:"primary"`
	Size     int    `json:"size"`
}

type ModrinthProjectSummary struct {
	ModrinthProject

	ID                string               `json:"project_id"`
	IconURL           string               `json:"icon_url"`
	Color             int                  `json:"color"`
	ThreadID          string               `json:"thread_id"`
	Monetization      ModrinthMonetization `json:"monetization_status"`
	Author            string               `json:"author"`
	DisplayCategories []string             `json:"display_categories"`
	GameVersions      []string             `json:"versions"`
	Followers         int                  `json:"follows"`
	Created           string               `json:"date_created"`
	Updated           string               `json:"date_modified"`
	LatestGameVersion string               `json:"latest_version"`
	License           string               `json:"license"`
	FeaturedGallery   string               `json:"featured_gallery"`
	Dependencies      []string             `json:"dependencies"`
}

type ModrinthSummary struct {
	Hits []ModrinthProjectSummary `json:"hits"`

	Offset int `json:"offset"`
	Limit  int `json:"limit"`
	Total  int `json:"total_hits"`
}

type Modrinth struct {
	bucket.HttpRepository
	bucket.LockRepository

	Context *bucket.OpenContext
}

func init() {
	bucket.RegisterRepository(MODRINTH_REPOSITORY,
		func(ctx context.Context, oc *bucket.OpenContext, opts map[string]string) bucket.Repository {
			return NewModrinthRepository(ctx, oc) // Go boilerplate
		})
}

func NewModrinthRepository(lock context.Context, context *bucket.OpenContext) *Modrinth {
	return &Modrinth{
		HttpRepository: *bucket.NewHttpRepository(MODRINTH_ENDPOINT),
		LockRepository: bucket.LockRepository{Lock: lock},
		Context:        context,
	}
}

func (r *Modrinth) makreq() *resty.Request {
	return r.HttpClient.R().SetContext(r.Lock)
}

func (r *Modrinth) Resolve(plugin bucket.Plugin) (bucket.RemotePlugin, []bucket.RemotePlugin, error) {
	if loc, ok := plugin.(bucket.LocalPlugin); ok {
		h := sha1.New()
		if _, err := io.Copy(h, loc.File); err != nil {
			return nil, nil, err
		}

		ver, err := r.GetByHash(hex.EncodeToString(h.Sum(nil)))
		if err == nil {
			res := ver.(*ModrinthVersion).ModrinthProject
			return res, []bucket.RemotePlugin{res}, nil
		} // else try to resolve by name
	}

	res, tot, err := r.Search(plugin.GetName(), 5)
	if err != nil {
		return nil, nil, err
	}

	if tot == 0 {
		return nil, nil, parseError(fmt.Errorf("no match found for \"%s\"", plugin.GetName()))
	}

	return res[0], res, nil
}

func (r *Modrinth) Get(identifier string) (bucket.RemotePlugin, error) {
	var plugin ModrinthProject

	res, err := r.makreq().SetResult(&plugin).Get("/project/" + identifier)
	if err != nil {
		return nil, err
	}

	if res.StatusCode() != 200 {
		return nil, parseReqError(res)
	}

	proj := res.Result().(*ModrinthProject)
	proj.repository = r

	return proj, nil
}

func (r *Modrinth) GetByHash(sha1 string) (bucket.RemoteVersion, error) {
	var plugin ModrinthVersion

	res, err := r.makreq().SetResult(&plugin).Get("/version_file/" + sha1)
	if err != nil {
		return nil, err
	}

	if res.StatusCode() != 200 {
		return nil, parseReqError(res)
	}

	ver := res.Result().(*ModrinthVersion)

	prj, err := r.Get(ver.ProjectID)
	if err != nil {
		return nil, fmt.Errorf("%v: version file found but associated project is unavailable", err)
	}

	ver.ModrinthProject = *(prj.(*ModrinthProject))

	return ver, nil
}

func (r *Modrinth) search(options map[string]string, max int) ([]bucket.RemotePlugin, int, error) {
	var result ModrinthSummary

	if max > 0 {
		options["limit"] = strconv.Itoa(max)
	}

	res, err := r.makreq().
		SetQueryParams(options).
		SetResult(&result).
		Get("/search")

	if err != nil {
		return nil, -1, parseError(err)
	}

	if res.StatusCode() != 200 {
		return nil, -1, parseReqError(res)
	}

	summary := res.Result().(*ModrinthSummary)

	var versions []bucket.RemotePlugin
	for i := range summary.Hits {
		summary.Hits[i].repository = r
		versions = append(versions, &summary.Hits[i])
	}

	return versions, summary.Total, nil
}

func (r *Modrinth) SearchAll(query string, max int) ([]bucket.RemotePlugin, int, error) {
	return r.search(map[string]string{
		"query": query,
	}, max)
}

func (r *Modrinth) Search(query string, max int) ([]bucket.RemotePlugin, int, error) {
	loaders := r.Context.Platform.Type().EveryCompatible()
	for i, v := range loaders {
		loaders[i] = fmt.Sprintf("\"categories:%s\"", v)
	}

	return r.search(map[string]string{
		"query":  query,
		"facets": fmt.Sprintf("[[\"categories:%s\"]]", r.Context.PlatformName()),
	}, max)
}

func (s *ModrinthProjectSummary) UnmarshalJSON(data []byte) error {
	type Alias ModrinthProjectSummary
	if err := json.Unmarshal(data, (*Alias)(s)); err != nil {
		return err
	}

	s.ModrinthProject.ID = s.ID
	s.ModrinthProject.Versions = s.GameVersions
	s.ModrinthProject.Followers = s.Followers
	s.ModrinthProject.Published = s.Created
	s.ModrinthProject.Updated = s.Updated
	s.ModrinthProject.License.ID = s.License

	return nil
}

func (r *Modrinth) GetVersion(identifier string) (bucket.RemoteVersion, error) {
	var version ModrinthVersion

	res, err := r.makreq().SetResult(&version).Get("/project/version/" + identifier)
	if err != nil {
		return nil, parseError(err)
	}

	if res.StatusCode() != 200 {
		return nil, parseReqError(res)
	}

	return res.Result().(*ModrinthVersion), nil
}

func (p ModrinthProject) GetName() string {
	return p.Title
}

func (p ModrinthProject) GetLatestVersion() (bucket.RemoteVersion, error) {
	vers, err := p.GetVersions()
	if err != nil {
		return nil, err
	}

	return vers[0], nil
}

func (p ModrinthProject) GetVersion(identifier string) (bucket.RemoteVersion, error) {
	return p.repository.GetVersion(identifier)
}

func (p ModrinthProject) GetVersions() ([]bucket.RemoteVersion, error) {
	var versions []ModrinthVersion

	res, err := p.repository.HttpClient.R().SetResult(&versions).Get("/project/" + p.Slug + "/version")
	if err != nil {
		return nil, parseError(err)
	}

	if res.StatusCode() != 200 {
		return nil, parseReqError(res)
	}

	var remoteVersions []bucket.RemoteVersion
	for i := range versions {
		versions[i].ModrinthProject = p
		remoteVersions = append(remoteVersions, versions[i])
	}

	return remoteVersions, nil
}

func (p ModrinthProject) GetVersionIdentifiers() ([]string, error) {
	return p.Versions, nil
}

func (p ModrinthProject) Compatible(platform bucket.PlatformType) bool {
	latest, err := p.GetLatestVersion()
	if err != nil {
		return false // If we fail to retrieve a version we can't be sure it's compatible
	}

	return latest.Compatible(platform)
}

func (p ModrinthProject) GetIdentifier() string {
	return p.Slug
}

func (p ModrinthProject) GetAuthors() []string {
	return []string{p.Team}
}

func (p ModrinthProject) GetDescription() string {
	return p.Body
}

func (p ModrinthProject) GetWebsite() string {
	return p.WikiUrl
}

func (p ModrinthVersion) GetIdentifier() string {
	return p.ID
}

func (p ModrinthVersion) GetName() string {
	return p.Name
}

func (p ModrinthVersion) GetDependencies() []bucket.Dependency {
	panic("not implemented") // TODO: Implement
}

func (p ModrinthVersion) Compatible(platform bucket.PlatformType) bool {
	return slices.Contains(p.Loaders, platform.Name)
}

func (p ModrinthVersion) GetFiles() ([]bucket.RemoteFile, error) {
	var remoteFiles []bucket.RemoteFile
	for i := range p.Files {
		remoteFiles = append(remoteFiles, &p.Files[i])
	}

	return remoteFiles, nil
}

func (f *ModrinthFile) Name() string {
	return f.Filename
}

func (f *ModrinthFile) Optional() bool {
	return !f.Primary
}

func (f *ModrinthFile) Download() (io.ReadCloser, error) {
	req := f.repository.HttpClient.R()
	req.SetDoNotParseResponse(true)

	resp, err := req.Get(f.URL)
	if err != nil {
		return nil, parseError(err)
	}

	if resp.StatusCode() != 200 {
		return nil, parseReqError(resp)
	}

	raw := resp.RawBody()

	f.hasher = sha512.New()
	hashed := io.TeeReader(raw, f.hasher)

	reader := struct {
		io.Reader
		io.Closer
	}{hashed, raw} // TeeReader strips away the closer aspect

	return reader, nil
}

func (f *ModrinthFile) Verify() error {
	if f.hasher == nil {
		return parseError(errors.New("file not downloaded"))
	}

	hash := hex.EncodeToString(f.hasher.Sum(nil))
	if hash != f.Hashes.Sha512 {
		return parseError(errors.New("hash mismatch"))
	}

	return nil
}

func parseReqError(res *resty.Response) error {
	var err map[string]string
	json.Unmarshal(res.Body(), &err)

	return fmt.Errorf("modrinth: error %s: %s", res.Status(), err["error"])
}

func parseError(err error) error {
	return fmt.Errorf("modrinth: %s", err)
}
