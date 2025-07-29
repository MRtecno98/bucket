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
	"strconv"
	"strings"

	"github.com/MRtecno98/bucket/bucket"
	"github.com/go-resty/resty/v2"
	"golang.org/x/exp/slices"
)

// TODO: Modrinth repository format (https://modrinth.com/api/docs)

const ModrinthEndpoint = "https://api.modrinth.com/v2"

const ModrinthRepository = "modrinth"

type ModrinthSide string
type ModrinthType string
type ModrinthDependency string
type ModrinthVersionType string
type ModrinthMonetization string

const (
	SideRequired    ModrinthSide = "required"
	SideOptional    ModrinthSide = "optional"
	SideUnsupported ModrinthSide = "unsupported"

	ProjectMod          ModrinthType = "mod"
	ProjectModpack      ModrinthType = "modpack"
	ProjectResourcepack ModrinthType = "resourcepack"

	DependencyRequired ModrinthDependency = "required"
	DependencyOptional ModrinthDependency = "optional"
	DependencyIncompat ModrinthDependency = "incompatible"
	DependencyEmbedded ModrinthDependency = "embedded"

	ModrinthRelease ModrinthVersionType = "release"
	ModrinthBeta    ModrinthVersionType = "beta"
	ModrinthAlpha   ModrinthVersionType = "alpha"

	ModrinthMonetized        ModrinthMonetization = "monetized"
	ModrinthDemonetized      ModrinthMonetization = "demonetized"
	ModrinthForceDemonetized ModrinthMonetization = "force-demonetized"
)

type ModrinthProject struct {
	repository *Modrinth `json:"-"`

	authors []ModrinthMember

	ID            string       `json:"id"`
	Slug          string       `json:"slug"`
	Title         string       `json:"title"`
	Description   string       `json:"description"`
	Categories    []string     `json:"categories"`
	ClientSide    ModrinthSide `json:"client_side"`
	ServerSide    ModrinthSide `json:"server_side"`
	Body          string       `json:"body"`
	AdtCategories []string     `json:"additional_categories"`
	IssuesURL     string       `json:"issues_url"`
	SourceURL     string       `json:"source_url"`
	WikiURL       string       `json:"wiki_url"`
	DiscordURL    string       `json:"discord_url"`
	ProjectType   ModrinthType `json:"project_type"`
	Downloads     int          `json:"downloads"`
	IconURL       string       `json:"icon_url"`
	Team          string       `json:"team"`
	Published     string       `json:"published"`
	Updated       string       `json:"updated"`
	Followers     int          `json:"followers"`
	Versions      []string     `json:"versions"`
	Gallery       []struct {
		URL         string `json:"url"`
		RawURL      string `json:"raw_url"`
		Featured    bool   `json:"featured"`
		Title       string `json:"title"`
		Description string `json:"description"`
		Created     string `json:"created"`
		Ordering    int    `json:"ordering"`
	} `json:"gallery"`

	License struct {
		ID   string `json:"id"`
		Name string `json:"name"`
		URL  string `json:"url"`
	} `json:"license"`

	DonationUrls []struct {
		ID       string `json:"id"`
		Platform string `json:"platform"`
		URL      string `json:"url"`
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

type ModrinthMember struct {
	ModrinthUser `json:"user"`

	TeamID      string `json:"team_id"`
	Role        string `json:"role"`
	Permissions int    `json:"permissions"`
	Accepted    bool   `json:"accepted"`
	Ordering    int    `json:"ordering"`
}

type ModrinthUser struct {
	ID        string `json:"id"`
	Username  string `json:"username"`
	AvatarURL string `json:"avatar_url"`
	Bio       string `json:"bio"`
	Created   string `json:"date_created"`
	Role      string `json:"role"`
	Badges    int    `json:"badges"`
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
	Gallery           []string             `json:"gallery"`
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
	bucket.HTTPRepository
	bucket.LockRepository

	Context *bucket.OpenContext
}

func init() {
	bucket.RegisterRepository(ModrinthRepository,
		func(ctx context.Context, oc *bucket.OpenContext, opts map[string]string) bucket.Repository {
			return NewModrinthRepository(ctx, oc) // Go boilerplate
		})
}

func NewModrinthRepository(lock context.Context, context *bucket.OpenContext) *Modrinth {
	return &Modrinth{
		HTTPRepository: *bucket.NewHTTPRepository(ModrinthEndpoint),
		LockRepository: bucket.LockRepository{Lock: lock},
		Context:        context,
	}
}

func (r *Modrinth) Provider() string {
	return ModrinthRepository
}

func (r *Modrinth) makreq() *resty.Request {
	return r.HTTPClient.R().SetContext(r.Lock)
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
			return &res, []bucket.RemotePlugin{&res}, nil
		} // else try to resolve by name
	}

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

	if tot == 0 {
		return nil, nil, r.parseError(fmt.Errorf("no match found for \"%s\"", plugin.GetName()))
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
		return nil, r.parseReqError(res)
	}

	proj := res.Result().(*ModrinthProject)
	proj.repository = r

	return proj, proj.requestMembers()
}

func (r *Modrinth) GetByHash(sha1 string) (bucket.RemoteVersion, error) {
	var plugin ModrinthVersion

	res, err := r.makreq().SetResult(&plugin).Get("/version_file/" + sha1)
	if err != nil {
		return nil, err
	}

	if res.StatusCode() != 200 {
		return nil, r.parseReqError(res)
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
		return nil, -1, r.parseError(err)
	}

	if res.StatusCode() != 200 {
		return nil, -1, r.parseReqError(res)
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
	qmap := map[string]string{
		"query": query,
	}

	if r.Context.Platform != nil {
		loaders := r.Context.Platform.Type().EveryCompatible()
		for i, v := range loaders {
			loaders[i] = fmt.Sprintf("\"categories:%s\"", v)
		}

		qmap["facets"] = fmt.Sprintf("[[%s]]", strings.Join(loaders, ", "))
	}

	return r.search(qmap, max)
}

func (r *Modrinth) parseReqError(res *resty.Response) error {
	var err map[string]string
	json.Unmarshal(res.Body(), &err)

	return fmt.Errorf("modrinth: error %s: %s", res.Status(), err["error"])
}

func (r *Modrinth) parseError(err error) error {
	return fmt.Errorf("modrinth: %s", err)
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

func (s *ModrinthProjectSummary) GetAuthors() []string {
	if len(s.authors) != 0 {
		return s.ModrinthProject.GetAuthors()
	}

	return []string{s.Author}
}

func (r *Modrinth) GetVersionByID(identifier string) (bucket.RemoteVersion, error) {
	var version ModrinthVersion

	res, err := r.makreq().SetResult(&version).Get("/project/version/" + identifier)
	if err != nil {
		return nil, r.parseError(err)
	}

	if res.StatusCode() != 200 {
		return nil, r.parseReqError(res)
	}

	return res.Result().(*ModrinthVersion), nil
}

func (p *ModrinthProject) GetName() string {
	return p.Title
}

func (p *ModrinthProject) GetRepository() bucket.Repository {
	return p.repository
}

func (p *ModrinthProject) GetLatestVersion() (bucket.RemoteVersion, error) {
	vers, err := p.GetVersions(1)
	if err != nil {
		return nil, err
	}

	return vers[0], nil
}

func (p *ModrinthProject) GetVersionByID(identifier string) (bucket.RemoteVersion, error) {
	return p.repository.GetVersionByID(identifier)
}

func (p *ModrinthProject) GetVersions(limit int) ([]bucket.RemoteVersion, error) {
	var versions []ModrinthVersion

	res, err := p.repository.HTTPClient.R().SetResult(&versions).Get("/project/" + p.Slug + "/version")
	if err != nil {
		return nil, p.repository.parseError(err)
	}

	if res.StatusCode() != 200 {
		return nil, p.repository.parseReqError(res)
	}

	var remoteVersions []bucket.RemoteVersion
	for i := range versions {
		if limit > 0 && i >= limit {
			break
		}

		versions[i].ModrinthProject = *p
		remoteVersions = append(remoteVersions, &versions[i])
	}

	return remoteVersions, nil
}

func (p *ModrinthProject) GetVersionIdentifiers() ([]string, error) {
	return p.Versions, nil
}

func (p *ModrinthProject) GetLatestCompatible(platform bucket.PlatformType) (bucket.RemoteVersion, error) {
	versions, err := p.GetVersions(0)
	if err != nil {
		return nil, err
	}

	for _, v := range versions {
		if v.Compatible(platform) {
			return v, nil
		}
	}

	return nil, p.repository.parseError(fmt.Errorf("no compatible version found"))
}

func (p ModrinthProject) Compatible(platform bucket.PlatformType) bool {
	ver, err := p.GetLatestCompatible(platform)
	return err == nil && ver != nil
}

func (p *ModrinthProject) GetDependencies() []bucket.Dependency {
	panic("not implemented") // TODO: Implement
}

func (p *ModrinthProject) requestMembers() error {
	res, err := p.repository.makreq().
		SetResult(&p.authors).
		Get("/project/" + p.ID + "/members")
	if err != nil {
		return p.repository.parseError(err)
	}

	if res.StatusCode() != 200 {
		return p.repository.parseReqError(res)
	}

	return nil
}

func (p *ModrinthProject) GetAuthors() []string {
	if len(p.authors) == 0 {
		if err := p.requestMembers(); err != nil {
			return []string{}
		}
	}

	var authors []string
	for _, v := range p.authors {
		authors = append(authors, v.Username)
	}

	return authors
}

func (p *ModrinthProject) GetIdentifier() string {
	return p.Slug
}

func (p *ModrinthProject) GetDescription() string {
	return p.Body
}

func (p *ModrinthProject) GetWebsite() string {
	return p.WikiURL
}

func (p *ModrinthVersion) GetVersion() string {
	return p.VersionNumber
}

func (p *ModrinthVersion) GetVersionName() string {
	return p.Name
}

func (p *ModrinthVersion) GetDependencies() []bucket.Dependency {
	panic("not implemented") // TODO: Implement
}

func (p *ModrinthVersion) Compatible(platform bucket.PlatformType) bool {
	comp := platform.EveryCompatible()
	for _, v := range p.Loaders {
		if slices.Contains(comp, v) {
			return true
		}
	}

	return false
}

func (p *ModrinthVersion) GetFiles() ([]bucket.RemoteFile, error) {
	var remoteFiles []bucket.RemoteFile
	for i := range p.Files {
		if p.Files[i].repository == nil {
			p.Files[i].repository = p.repository
		}

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
	req := f.repository.HTTPClient.R()
	req.SetDoNotParseResponse(true)

	resp, err := req.Get(f.URL)
	if err != nil {
		return nil, f.repository.parseError(err)
	}

	if resp.StatusCode() != 200 {
		return nil, f.repository.parseReqError(resp)
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
		return f.repository.parseError(errors.New("file not downloaded"))
	}

	hash := hex.EncodeToString(f.hasher.Sum(nil))
	if hash != f.Hashes.Sha512 {
		return f.repository.parseError(errors.New("hash mismatch"))
	}

	return nil
}
