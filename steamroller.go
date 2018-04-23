package steamroller

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	yamlpatch "github.com/krishicks/yaml-patch"
	yaml "gopkg.in/yaml.v2"
)

// interpreter represents the information required to invoke an interpreter.
type interpreter struct {
	// The path to the interpreter.
	Path string
	// The args used when invoking the interpreter with a script as an arg.
	Args []string
	// The template to use when inlining the script contents
	Template string
}

// interpreters maps file extensions to interpreters.
var interpreters = map[string]interpreter{
	"":    {"sh", []string{"-c"}, shTemplate},
	".sh": {"sh", []string{"-c"}, shTemplate},
	".rb": {"ruby", []string{"-e"}, ""},
	".py": {"python", []string{"-c"}, ""},
	".js": {"node", []string{"-e"}, ""},
}

type Config struct {
	ResourceMap map[string]string `yaml:"resource_map"`
}

func Steamroll(filemap map[string]string, pipelineBytes []byte) ([]byte, error) {
	var pipeline map[string]interface{}
	err := yaml.Unmarshal(pipelineBytes, &pipeline)
	if err != nil {
		log.Fatalf("failed to unmarshal pipeline: %s", err)
	}

	files, err := findFiles(pipeline["jobs"])
	if err != nil {
		log.Fatalf("failed to find files: %s", err)
	}

	var patch yamlpatch.Patch
	for file := range files {
		if !resourceIsMapped(filemap, file) {
			continue
		}

		bs, err := loadBytes(filemap, file)
		if err != nil {
			log.Fatalf("failed to load yml bytes: %s", err)
		}

		var bi map[interface{}]interface{}
		err = yaml.Unmarshal(bs, &bi)
		if err != nil {
			log.Fatalf("failed to unmarshal: %s", err)
		}

		patch = append(patch, yamlpatch.Operation{
			Op:    yamlpatch.OpAdd,
			Path:  yamlpatch.OpPath(fmt.Sprintf("/jobs/file=%s/config", strings.Replace(file, "/", "~1", -1))),
			Value: yamlpatch.NewNodeFromMap(bi),
		})
		patch = append(patch, yamlpatch.Operation{
			Op:   yamlpatch.OpRemove,
			Path: yamlpatch.OpPath(fmt.Sprintf("/jobs/file=%s/file", strings.Replace(file, "/", "~1", -1))),
		})
	}

	bs, err := patch.Apply(pipelineBytes)
	if err != nil {
		log.Fatalf("failed to apply patch: %s", err)
	}

	patch = nil
	pipeline = map[string]interface{}{}
	err = yaml.Unmarshal(bs, &pipeline)
	if err != nil {
		log.Fatalf("failed to unmarshal pipeline: %s", err)
	}

	files, err = findRunPaths(pipeline["jobs"])
	if err != nil {
		log.Fatalf("failed to find files: %s", err)
	}

	for file := range files {
		if !resourceIsMapped(filemap, file) {
			continue
		}

		bs, err := loadBytes(filemap, file)
		if err != nil {
			log.Fatalf("failed to load sh bytes: %s", err)
		}

		interpreter := interpreters[filepath.Ext(file)]

		var script string
		if interpreter.Template != "" {
			s := inlineScript{
				Contents: string(bs),
			}

			buf := &bytes.Buffer{}
			tmpl := template.Must(template.New("run").Parse(interpreter.Template))
			err = tmpl.Execute(buf, s)
			if err != nil {
				log.Fatalf("failed to execute template: %s", err)
			}

			script = buf.String()
		} else {
			script = string(bs)
		}

		args := []string{}
		args = append(args, interpreter.Args...)
		args = append(args, script)

		patch = append(patch, yamlpatch.Operation{
			Op:   yamlpatch.OpReplace,
			Path: yamlpatch.OpPath(fmt.Sprintf("/jobs/path=%s", strings.Replace(file, "/", "~1", -1))),
			Value: yamlpatch.NewNodeFromMap(map[interface{}]interface{}{
				"path": interpreter.Path,
				"args": args,
			}),
		})
	}

	if patch != nil {
		return patch.Apply(bs)
	}

	return bs, nil
}

func findFiles(data interface{}) (map[string]struct{}, error) {
	files := map[string]struct{}{}

	switch i1 := data.(type) {
	case map[interface{}]interface{}:
		if _, hasTask := i1["task"]; hasTask {
			if path, hasFile := i1["file"]; hasFile {
				if pathStr, ok := path.(string); ok {
					files[pathStr] = struct{}{}
				}
			}
		}
		for _, v := range i1 {
			newFiles, err := findFiles(v)
			if err != nil {
				log.Fatalf("failed to find files: %s", err)
			}
			for k := range newFiles {
				files[k] = struct{}{}
			}
		}
	case []interface{}:
		for _, v := range i1 {
			newFiles, err := findFiles(v)
			if err != nil {
				log.Fatalf("failed to find files: %s", err)
			}
			for k := range newFiles {
				files[k] = struct{}{}
			}
		}
	}

	return files, nil
}

func findRunPaths(data interface{}) (map[string]struct{}, error) {
	files := map[string]struct{}{}

	switch i1 := data.(type) {
	case map[interface{}]interface{}:
		if iface, hasRun := i1["run"]; hasRun {
			if path, hasPath := iface.(map[interface{}]interface{})["path"]; hasPath {
				if pathStr, ok := path.(string); ok {
					files[pathStr] = struct{}{}
				}
			}
		}
		for _, v := range i1 {
			newFiles, err := findRunPaths(v)
			if err != nil {
				log.Fatalf("failed to find files: %s", err)
			}
			for k := range newFiles {
				files[k] = struct{}{}
			}
		}
	case []interface{}:
		for _, v := range i1 {
			newFiles, err := findRunPaths(v)
			if err != nil {
				log.Fatalf("failed to find files: %s", err)
			}
			for k := range newFiles {
				files[k] = struct{}{}
			}
		}
	}

	return files, nil
}

func resourceIsMapped(resourceMap map[string]string, path string) bool {
	resourceRoot := strings.Split(path, string(os.PathSeparator))[0]
	_, ok := resourceMap[resourceRoot]
	return ok
}

func loadBytes(resourceMap map[string]string, path string) ([]byte, error) {
	resourceRoot := strings.Split(path, string(os.PathSeparator))[0]

	resourcePath, ok := resourceMap[resourceRoot]
	if !ok || resourcePath == "" {
		return nil, fmt.Errorf("no resource map provided for %s", path)
	}

	if strings.HasPrefix(resourcePath, "~") {
		resourcePath = filepath.Join(os.Getenv("HOME"), resourcePath[2:])
	}

	actualPath := filepath.Join(resourcePath, strings.Replace(path, resourceRoot, "", -1))

	return ioutil.ReadFile(actualPath)
}

type inlineScript struct {
	Contents string
}

const shTemplate = `cat > task.sh <<'EO_SH'
{{.Contents}}
EO_SH

chmod +x task.sh
./task.sh
`
