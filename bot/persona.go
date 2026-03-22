package bot

import (
	"os"
	"strings"
)

// Persona represents a loaded character persona from a markdown file.
type Persona struct {
	Name        string
	Personality string
	Strategy    string
	Catchphrase string
	Raw         string
}

// LoadPersona reads and parses a persona markdown file.
// Format:
//
//	# Character Name
//	## 性格
//	...
//	## 戦略
//	...
//	## 口癖
//	...
func LoadPersona(path string) (*Persona, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	raw := string(data)
	p := &Persona{Raw: raw}

	lines := strings.Split(raw, "\n")
	var currentSection string
	var sectionBuf strings.Builder

	flushSection := func() {
		content := strings.TrimSpace(sectionBuf.String())
		switch currentSection {
		case "name":
			p.Name = content
		case "性格", "personality":
			p.Personality = content
		case "戦略", "strategy":
			p.Strategy = content
		case "口癖", "catchphrase":
			p.Catchphrase = content
		}
		sectionBuf.Reset()
	}

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "# ") && !strings.HasPrefix(trimmed, "## ") {
			flushSection()
			currentSection = "name"
			sectionBuf.WriteString(strings.TrimPrefix(trimmed, "# "))
			continue
		}
		if strings.HasPrefix(trimmed, "## ") {
			flushSection()
			currentSection = strings.ToLower(strings.TrimPrefix(trimmed, "## "))
			continue
		}
		if currentSection != "" {
			sectionBuf.WriteString(line)
			sectionBuf.WriteString("\n")
		}
	}
	flushSection()

	if p.Name == "" {
		p.Name = "LLM"
	}

	return p, nil
}
