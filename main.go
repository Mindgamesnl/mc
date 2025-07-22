package main

import (
	"bufio"
	"fmt"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"gopkg.in/yaml.v3"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

var Version = "dev" // Set by build flags

type Config struct {
	Version string `yaml:"version"`
	Memory  string `yaml:"memory"`
	Port    int    `yaml:"port"`
}

type item struct {
	title string
	desc  string
}

func (i item) FilterValue() string { return i.title }
func (i item) Title() string       { return i.title }
func (i item) Description() string { return i.desc }

type model struct {
	list     list.Model
	choice   string
	quitting bool
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.list.SetWidth(msg.Width)
		return m, nil

	case tea.KeyMsg:
		switch keypress := msg.String(); keypress {
		case "ctrl+c", "q":
			m.quitting = true
			return m, tea.Quit

		case "enter":
			i, ok := m.list.SelectedItem().(item)
			if ok {
				m.choice = i.title
			}
			return m, tea.Quit
		}
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m model) View() string {
	if m.choice != "" {
		return ""
	}
	if m.quitting {
		return "Bye!\n"
	}
	return "\n" + m.list.View()
}

func main() {
	if len(os.Args) > 1 && os.Args[1] == "test" {
		testJavaSetup()
		return
	}

	if len(os.Args) > 1 && (os.Args[1] == "version" || os.Args[1] == "--version" || os.Args[1] == "-v") {
		fmt.Printf("mc version %s\n", Version)
		return
	}

	if err := validateJava(); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	config, exists, err := loadConfig()
	if err != nil {
		fmt.Printf("Error loading config: %v\n", err)
		os.Exit(1)
	}

	var version string
	if len(os.Args) > 1 {
		version = os.Args[1]
	}

	if !exists && version == "" {
		fmt.Print("No mc.yml found. Enter Minecraft version (e.g., 1.21.4): ")
		scanner := bufio.NewScanner(os.Stdin)
		if scanner.Scan() {
			version = strings.TrimSpace(scanner.Text())
		}
		if version == "" {
			fmt.Println("No version specified")
			os.Exit(1)
		}
	}

	if version != "" {
		if !isValidVersion(version) {
			fmt.Printf("Invalid version format: %s\n", version)
			os.Exit(1)
		}

		config = &Config{
			Version: version,
			Memory:  "2G",
			Port:    25565,
		}

		if err := saveConfig(config); err != nil {
			fmt.Printf("Error saving config: %v\n", err)
			os.Exit(1)
		}
	} else if exists {
		versions := getVersionsFromJars()
		if len(versions) > 1 {
			selected, err := selectVersion(versions)
			if err != nil {
				fmt.Printf("Error selecting version: %v\n", err)
				os.Exit(1)
			}
			config.Version = selected
		}
	}

	jarPath := fmt.Sprintf("paper-%s.jar", config.Version)

	if _, err := os.Stat(jarPath); os.IsNotExist(err) {
		fmt.Printf("Downloading Paper %s...\n", config.Version)
		if err := downloadPaper(config.Version, jarPath); err != nil {
			fmt.Printf("Error downloading Paper: %v\n", err)
			os.Exit(1)
		}
	}

	fmt.Printf("Starting Minecraft server %s...\n", config.Version)
	if err := runServer(jarPath, config.Memory); err != nil {
		fmt.Printf("Error running server: %v\n", err)
		os.Exit(1)
	}
}

func validateJava() error {
	cmd := exec.Command("java", "-version")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("java not found in PATH, please install Java")
	}
	return nil
}

func loadConfig() (*Config, bool, error) {
	data, err := os.ReadFile("mc.yml")
	if os.IsNotExist(err) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, false, err
	}

	return &config, true, nil
}

func saveConfig(config *Config) error {
	data, err := yaml.Marshal(config)
	if err != nil {
		return err
	}
	return os.WriteFile("mc.yml", data, 0644)
}

func isValidVersion(version string) bool {
	pattern := `^\d+\.\d+(\.\d+)?$`
	matched, _ := regexp.MatchString(pattern, version)
	return matched
}

func getVersionsFromJars() []string {
	files, err := filepath.Glob("paper-*.jar")
	if err != nil {
		return []string{}
	}

	versions := make([]string, 0, len(files))
	for _, file := range files {
		if strings.HasPrefix(file, "paper-") && strings.HasSuffix(file, ".jar") {
			version := strings.TrimPrefix(file, "paper-")
			version = strings.TrimSuffix(version, ".jar")
			versions = append(versions, version)
		}
	}

	sort.Slice(versions, func(i, j int) bool {
		return compareVersions(versions[i], versions[j]) > 0
	})

	return versions
}

func compareVersions(v1, v2 string) int {
	parts1 := strings.Split(v1, ".")
	parts2 := strings.Split(v2, ".")

	maxLen := len(parts1)
	if len(parts2) > maxLen {
		maxLen = len(parts2)
	}

	for i := 0; i < maxLen; i++ {
		var p1, p2 int
		if i < len(parts1) {
			p1, _ = strconv.Atoi(parts1[i])
		}
		if i < len(parts2) {
			p2, _ = strconv.Atoi(parts2[i])
		}

		if p1 > p2 {
			return 1
		} else if p1 < p2 {
			return -1
		}
	}
	return 0
}

func selectVersion(versions []string) (string, error) {
	items := make([]list.Item, len(versions))
	for i, version := range versions {
		items[i] = item{title: version, desc: fmt.Sprintf("Paper %s", version)}
	}

	l := list.New(items, list.NewDefaultDelegate(), 0, 0)
	l.Title = "Select Minecraft Version"
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)
	l.Styles.Title = lipgloss.NewStyle().MarginLeft(2)

	m := model{list: l}

	p := tea.NewProgram(m)
	finalModel, err := p.Run()
	if err != nil {
		return "", err
	}

	if finalModel.(model).quitting {
		os.Exit(0)
	}

	return finalModel.(model).choice, nil
}

func downloadPaper(version, jarPath string) error {
	url := fmt.Sprintf("https://api.papermc.io/v2/projects/paper/versions/%s/builds", version)

	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("version %s not found", version)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	buildNumber := extractLatestBuild(string(body))
	if buildNumber == "" {
		return fmt.Errorf("no builds found for version %s", version)
	}

	downloadURL := fmt.Sprintf("https://api.papermc.io/v2/projects/paper/versions/%s/builds/%s/downloads/paper-%s-%s.jar",
		version, buildNumber, version, buildNumber)

	resp, err = http.Get(downloadURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("download failed with status: %d", resp.StatusCode)
	}

	file, err := os.Create(jarPath)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = io.Copy(file, resp.Body)
	return err
}

func extractLatestBuild(jsonBody string) string {
	pattern := `"build":(\d+)`
	re := regexp.MustCompile(pattern)
	matches := re.FindAllStringSubmatch(jsonBody, -1)

	if len(matches) == 0 {
		return ""
	}

	latestBuild := 0
	for _, match := range matches {
		if len(match) > 1 {
			if build, err := strconv.Atoi(match[1]); err == nil && build > latestBuild {
				latestBuild = build
			}
		}
	}

	if latestBuild > 0 {
		return strconv.Itoa(latestBuild)
	}
	return ""
}

func runServer(jarPath, memory string) error {
	// Auto-accept EULA
	if err := acceptEula(); err != nil {
		return fmt.Errorf("failed to accept EULA: %v", err)
	}

	cmd := exec.Command("java", fmt.Sprintf("-Xmx%s", memory), "-jar", jarPath, "nogui")
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

func acceptEula() error {
	eulaContent := "# Minecraft EULA\n# Auto-accepted by mc utility\neula=true\n"
	return os.WriteFile("eula.txt", []byte(eulaContent), 0644)
}

func testJavaSetup() {
	fmt.Println("Testing Java setup...")

	cmd := exec.Command("java", "-version")
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("❌ Java not found: %v\n", err)
		fmt.Println("Please install Java and add it to your PATH")
		os.Exit(1)
	}

	fmt.Printf("✅ Java is installed:\n%s", output)

	cmd = exec.Command("java", "-Xmx1G", "-version")
	err = cmd.Run()
	if err != nil {
		fmt.Printf("❌ Java memory allocation test failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("✅ Java memory allocation works correctly")
	fmt.Println("✅ Java setup is valid for running Minecraft servers")
}
