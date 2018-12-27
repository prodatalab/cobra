// Copyright © 2015 Steve Francia <spf@spf13.com>.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cmd

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/prodatalab/cobra"
	"github.com/spf13/viper"
)

var initCmd = &cobra.Command{
	Use:     "init [name]",
	Aliases: []string{"initialize", "initialise", "create"},
	Short:   "Initialize a Cobra Application",
	Long: `Initialize (cobra init) will create a new application, with a license
and the appropriate structure for a Cobra-based CLI application.

  * If a name is provided, it will be created in the current directory;
  * If no name is provided, the current directory will be assumed;
  * If a relative path is provided, it will be created inside $GOPATH
    (e.g. github.com/spf13/hugo);
  * If an absolute path is provided, it will be created;
  * If the directory already exists but is empty, it will be used.

Init will not use an existing directory with contents.`,

	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("INFO: In Run::x")
		wd, err := os.Getwd()
		if err != nil {
			er(err)
		}

		var project *Project
		if len(args) == 0 {
			project = NewProjectFromPath(wd)
		} else if len(args) == 1 {
			arg := args[0]
			if arg[0] == '.' {
				arg = filepath.Join(wd, arg)
			}
			if filepath.IsAbs(arg) {
				project = NewProjectFromPath(arg)
			} else {
				project = NewProject(arg)
			}
		} else {
			er("please provide only one argument")
		}

		initializeProject(project)

		fmt.Fprintln(cmd.OutOrStdout(), `Your Cobra application is ready at
`+project.AbsPath()+`

Give it a try by going there and running `+"`go run main.go`."+`
Add commands to it by running `+"`cobra add [cmdname]`.")
	},
}

func initializeProject(project *Project) {
	if !exists(project.AbsPath()) { // If path doesn't yet exist, create it
		err := os.MkdirAll(project.AbsPath(), os.ModePerm)
		if err != nil {
			er(err)
		}
	} else if !isEmpty(project.AbsPath()) { // If path exists and is not empty don't use it
		er("Cobra will not create a new project in a non empty directory: " + project.AbsPath())
	}

	// We have a directory and it's empty. Time to initialize it.
	createLicenseFile(project.License(), project.AbsPath())
	createMainFile(project)
	createRootCmdFile(project)
	createDockerfile(project)
	createReadme(project)
	createHelmCharts(project)
	createDroneFile(project)
}

func createLicenseFile(license License, path string) {
	data := make(map[string]interface{})
	data["copyright"] = copyrightLine()

	// Generate license template from text and data.
	text, err := executeTemplate(license.Text, data)
	if err != nil {
		er(err)
	}

	// Write license text to LICENSE file.
	err = writeStringToFile(filepath.Join(path, "LICENSE"), text)
	if err != nil {
		er(err)
	}
}

func createMainFile(project *Project) {
	mainTemplate := `{{ comment .copyright }}
{{if .license}}{{ comment .license }}{{end}}

package main

import "{{ .importpath }}"

func main() {
	cmd.Execute()
}
`
	data := project.ProjectToMap()

	mainScript, err := executeTemplate(mainTemplate, data)
	if err != nil {
		er(err)
	}

	err = writeStringToFile(filepath.Join(project.AbsPath(), "main.go"), mainScript)
	if err != nil {
		er(err)
	}
}

func createRootCmdFile(project *Project) {
	template := `{{comment .copyright}}
{{if .license}}{{comment .license}}{{end}}

package cmd

import (
	"fmt"
	"os"
{{if .viper}}
	homedir "github.com/mitchellh/go-homedir"{{end}}
	"github.com/prodatalab/cobra"{{if .viper}}
	"github.com/spf13/viper"{{end}}
){{if .viper}}

var cfgFile string{{end}}

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "{{.appName}}",
	Short: "A brief description of your application",
	Long: ` + "`" + `A longer description that spans multiple lines and likely contains
examples and usage of using your application. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.` + "`" + `,
	// Uncomment the following line if your bare application
	// has an action associated with it:
	//	Run: func(cmd *cobra.Command, args []string) { },
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() { {{- if .viper}}
	cobra.OnInitialize(initConfig)
{{end}}
	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.{{ if .viper }}
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.{{ .appName }}.yaml)"){{ else }}
	// rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.{{ .appName }}.yaml)"){{ end }}

	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}{{ if .viper }}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := homedir.Dir()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		// Search config in home directory with name ".{{ .appName }}" (without extension).
		viper.AddConfigPath(home)
		viper.SetConfigName(".{{ .appName }}")
	}

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		fmt.Println("Using config file:", viper.ConfigFileUsed())
	}
}{{ end }}
`

	data := project.ProjectToMap()
	data["viper"] = viper.GetBool("useViper")

	rootCmdScript, err := executeTemplate(template, data)
	if err != nil {
		er(err)
	}

	err = writeStringToFile(filepath.Join(project.CmdPath(), "root.go"), rootCmdScript)
	if err != nil {
		er(err)
	}

}

func createDockerfile(project *Project) {
	template := `
	FROM golang as builder

	ENV GO111MODULE=on

	WORKDIR /app

	COPY go.mod .
	COPY go.sum .

	RUN go mod download

	COPY . .

	RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build

	FROM scratch
	COPY --from=builder /app/{{.appName}} /app/
	EXPOSE 8080
	ENTRYPOINT ["/app/{{.appName}}"]
	`
	data := project.ProjectToMap()
	data[""] = viper.GetBool("useViper")
	dockerfile, err := executeTemplate(template, data)
	if err != nil {
		er(err)
	}

	err = writeStringToFile(filepath.Join(project.AbsPath(), "Dockerfile"), dockerfile)
	if err != nil {
		er(err)
	}
}

func createReadme(project *Project) {
	template := `
	{{.appName}}
	============
	`
	data := project.ProjectToMap()
	data[""] = viper.GetBool("useViper")
	readme, err := executeTemplate(template, data)
	if err != nil {
		er(err)
	}

	err = writeStringToFile(filepath.Join(project.AbsPath(), "README.md"), readme)
	if err != nil {
		er(err)
	}
}

func createHelmCharts(project *Project) {
	fmt.Println("Here I am")
	template := `helm create {{.appName}}`
	data := project.ProjectToMap()
	data[""] = viper.GetBool("useViper")
	path := project.AbsPath()
	fmt.Println("path: " + path)
	helmCmd, err := executeTemplate(template, data)
	if err != nil {
		er(err)
	}
	// fmt.Println("helmCmd: " + helmCmd + " absPath: " + project.AbsPath())
	var workDir string
	workDir, err = os.Getwd()
	if err != nil {
		er(err)
	}
	err = os.Chdir(project.AbsPath())
	if err != nil {
		er(err)
	}
	curDir, _ := os.Getwd()
	fmt.Println("working directory: " + curDir)
	cmdList := strings.Fields(helmCmd)
	cmd := exec.Command(cmdList[0], cmdList[1], cmdList[2])
	var out bytes.Buffer
	cmd.Stdout = &out
	err = cmd.Run()
	if err != nil {
		er(err)
	}
	fmt.Println("out: " + out.String())
	tmpList := strings.Split(project.Name(), "/")
	projectName := tmpList[len(tmpList)-1]
	fmt.Println("projectName: " + projectName)
	files, err := ioutil.ReadDir("./" + projectName)
	if err != nil {
		er(err)
	}
	for _, f := range files {
		os.Rename(projectName+"/"+f.Name(), f.Name())
	}
	os.RemoveAll("./" + projectName)

	// run helm manifests through the template engine
	customizeHelmCharts(project)

	// add requirements.yaml
	helmdeps := `
	# see helm docs
	# list your dependent charts here
	# dependencies:
	`
	err = ioutil.WriteFile("./requirements.yaml", []byte(helmdeps), 0644)
	if err != nil {
		er(err)
	}

	err = os.Chdir(workDir)
	if err != nil {
		er(err)
	}
}

func customizeHelmCharts(p *Project) {
	// load the values.yaml file
	values := viper.New()
	values.SetConfigName("values")
	values.AddConfigPath(".")
	err := values.ReadInConfig()
	if err != nil {
		er(err)
	}
	names := strings.Split(p.Name(), "/")
	name := names[len(names)-1]
	values.Set("image.repository", "prodatalab/"+name)
	values.WriteConfig()
}

func createDroneFile(p *Project) {
	droneText := `
kind: pipeline
name: default

steps:
- name: build
  image: golang
  commands:
  - go build
  - go test
`
	// data := project.ProjectToMap()
	// data[""] = viper.GetBool("useViper")
	// readme, err := executeTemplate(template, data)
	// if err != nil {
	//   er(err)
	// }

	err := writeStringToFile(filepath.Join(p.AbsPath(), ".drone.yml"), droneText)
	if err != nil {
		er(err)
	}
}
