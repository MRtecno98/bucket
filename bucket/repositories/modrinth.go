package repositories

import (
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"hash"
	"io"

	"github.com/MRtecno98/bucket/bucket"
	"github.com/go-resty/resty/v2"
	"golang.org/x/exp/slices"
)

// TODO: Modrinth repository format (https://modrinth.com/api/docs)

const MODRINTH_ENDPOINT = "https://api.modrinth.com/v2"

type ModrinthSide string
type ModrinthType string
type ModrinthDependency string
type ModrinthVersionType string

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

type Modrinth struct {
	bucket.HttpRepository
}

func NewModrinthRepository() *Modrinth {
	return &Modrinth{
		HttpRepository: *bucket.NewHttpRepository(MODRINTH_ENDPOINT),
	}
}

func (r *Modrinth) Resolve(plugin bucket.Plugin) (bucket.RemotePlugin, error) {
	panic("not implemented")
}

func (r *Modrinth) Get(identifier string) (bucket.RemotePlugin, error) {
	var plugin ModrinthProject

	res, err := r.HttpClient.R().SetResult(&plugin).Get("/project/" + identifier)
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

func (r *Modrinth) Search(query string) ([]bucket.RemotePlugin, error) {
	panic("not implemented")
}

func (r *Modrinth) SearchAll(query string) ([]bucket.RemotePlugin, error) {
	panic("not implemented")
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

func (p ModrinthProject) GetVersion(identifier string) (bucket.RemoteVersion, error) {
	panic("not implemented") // TODO: Implement
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
