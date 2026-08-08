package apps

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"os/exec"
	"strconv"
	"strings"

	"github.com/TrustEdgeOrg/TrustEdge-Agent/internal/identity"
)

// DockerContainer is a running/installed container observation.
type DockerContainer struct {
	ID     string
	Name   string
	Image  string
	Status string // running, exited, …
	Ports  []DockerPublishedPort
}

// DockerPublishedPort is a host↔container port mapping.
type DockerPublishedPort struct {
	HostIP        string
	HostPort      int
	ContainerPort int
	Protocol      string
}

// DockerLister lists Docker containers matching catalog image hints.
type DockerLister func(images []string) ([]DockerContainer, error)

// DockerDiscoverer finds known local model runtimes running as Docker containers.
type DockerDiscoverer struct {
	Log     *log.Logger
	Catalog *identity.Catalog
	ListFn  DockerLister
}

func catalogDockerImages(catalog *identity.Catalog) []string {
	if catalog == nil {
		catalog = identity.DefaultCatalog()
	}
	var out []string
	seen := make(map[string]struct{})
	for _, p := range catalog.Products() {
		for _, img := range p.DockerImages {
			img = strings.TrimSpace(strings.ToLower(img))
			if img == "" {
				continue
			}
			if _, ok := seen[img]; ok {
				continue
			}
			seen[img] = struct{}{}
			out = append(out, img)
		}
	}
	return out
}

// Discover returns ApplicationIdentity rows for matching Docker containers.
func (d *DockerDiscoverer) Discover() ([]identity.ApplicationIdentity, error) {
	if d == nil {
		return nil, nil
	}
	catalog := d.Catalog
	if catalog == nil {
		catalog = identity.DefaultCatalog()
	}
	images := catalogDockerImages(catalog)
	if len(images) == 0 {
		return nil, nil
	}
	listFn := d.ListFn
	if listFn == nil {
		listFn = listDockerContainers
	}
	containers, err := listFn(images)
	if err != nil {
		d.logf("docker discover: %v", err)
		return nil, nil // docker absent/unavailable is not fatal
	}
	var out []identity.ApplicationIdentity
	for _, c := range containers {
		if !containerImageMatches(c.Image, images) {
			continue
		}
		name := strings.TrimPrefix(c.Name, "/")
		path := fmt.Sprintf("docker://%s", firstNonEmpty(name, c.ID))
		id := identity.ApplicationIdentity{
			Path:              path,
			ResolvedPath:      path,
			Executable:        dockerExecutableHint(c.Image),
			ExecutablePath:    path,
			PackageManager:    "docker",
			PackageIdentifier: normalizeDockerImage(c.Image),
			Interpreter:       "docker",
			EntryPoint:        name,
		}
		// Encode published ports in EntryPoint? Better: stash via Version unused.
		// Ports are attached later in Engine from a side channel — store in PackageVersion
		// as opaque marker is wrong. Use a package-level lastDiscoverPorts map keyed by path.
		rememberDockerPorts(path, c)
		out = append(out, id)
	}
	return out, nil
}

func (d *DockerDiscoverer) logf(format string, args ...any) {
	if d != nil && d.Log != nil {
		d.Log.Printf(format, args...)
	}
}

var dockerPortsByPath = map[string]DockerContainer{}

func rememberDockerPorts(path string, c DockerContainer) {
	dockerPortsByPath[pathKey(path)] = c
}

func dockerContainerForPath(path string) (DockerContainer, bool) {
	c, ok := dockerPortsByPath[pathKey(path)]
	return c, ok
}

func dockerExecutableHint(image string) string {
	img := normalizeDockerImage(image)
	// ollama/ollama:latest → ollama
	if i := strings.LastIndex(img, "/"); i >= 0 && i+1 < len(img) {
		img = img[i+1:]
	}
	if i := strings.Index(img, ":"); i >= 0 {
		img = img[:i]
	}
	return img
}

func normalizeDockerImage(image string) string {
	image = strings.TrimSpace(strings.ToLower(image))
	// Strip digest.
	if i := strings.Index(image, "@"); i >= 0 {
		image = image[:i]
	}
	return image
}

func containerImageMatches(image string, catalogImages []string) bool {
	got := normalizeDockerImage(image)
	for _, want := range catalogImages {
		want = strings.TrimSpace(strings.ToLower(want))
		if want == "" {
			continue
		}
		if got == want || strings.HasPrefix(got, want+":") {
			return true
		}
		if strings.HasSuffix(got, "/"+want) || strings.Contains(got, "/"+want+":") {
			return true
		}
	}
	return false
}

// listDockerContainers runs `docker ps -a --format json` (one object per line).
func listDockerContainers(catalogImages []string) ([]DockerContainer, error) {
	_ = catalogImages
	cmd := exec.Command("docker", "ps", "-a", "--format", "{{json .}}")
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	return parseDockerPSJSON(out), nil
}

func parseDockerPSJSON(raw []byte) []DockerContainer {
	var out []DockerContainer
	for _, line := range bytes.Split(raw, []byte("\n")) {
		line = bytes.TrimSpace(line)
		if len(line) == 0 {
			continue
		}
		var row struct {
			ID     string `json:"ID"`
			Names  string `json:"Names"`
			Image  string `json:"Image"`
			Status string `json:"Status"`
			State  string `json:"State"`
			Ports  string `json:"Ports"`
		}
		if err := json.Unmarshal(line, &row); err != nil {
			continue
		}
		state := strings.ToLower(strings.TrimSpace(row.State))
		if state == "" {
			// Older format embeds state in Status ("Up 2 hours").
			status := strings.ToLower(row.Status)
			if strings.HasPrefix(status, "up") {
				state = "running"
			} else {
				state = "exited"
			}
		}
		c := DockerContainer{
			ID:     row.ID,
			Name:   firstNonEmpty(row.Names, row.ID),
			Image:  row.Image,
			Status: state,
			Ports:  parseDockerPorts(row.Ports),
		}
		out = append(out, c)
	}
	return out
}

// parseDockerPorts parses docker ps Ports field, e.g.
// "0.0.0.0:11434->11434/tcp, [::]:11434->11434/tcp"
func parseDockerPorts(s string) []DockerPublishedPort {
	var out []DockerPublishedPort
	for _, part := range strings.Split(s, ",") {
		part = strings.TrimSpace(part)
		if part == "" || !strings.Contains(part, "->") {
			continue
		}
		sides := strings.SplitN(part, "->", 2)
		if len(sides) != 2 {
			continue
		}
		hostSide := strings.TrimSpace(sides[0])
		contSide := strings.TrimSpace(sides[1])
		proto := "tcp"
		if i := strings.LastIndex(contSide, "/"); i >= 0 {
			proto = contSide[i+1:]
			contSide = contSide[:i]
		}
		contPort, _ := strconv.Atoi(contSide)
		hostIP := "0.0.0.0"
		hostPortStr := hostSide
		if i := strings.LastIndex(hostSide, ":"); i >= 0 {
			hostIP = hostSide[:i]
			hostPortStr = hostSide[i+1:]
			hostIP = strings.Trim(hostIP, "[]")
		}
		hostPort, err := strconv.Atoi(hostPortStr)
		if err != nil || hostPort <= 0 {
			continue
		}
		if hostIP == "" || hostIP == "*" {
			hostIP = "0.0.0.0"
		}
		out = append(out, DockerPublishedPort{
			HostIP:        hostIP,
			HostPort:      hostPort,
			ContainerPort: contPort,
			Protocol:      proto,
		})
	}
	return out
}

func newDockerDiscoverer(logger *log.Logger) Discoverer {
	return &DockerDiscoverer{
		Log:     logger,
		Catalog: identity.DefaultCatalog(),
	}
}
