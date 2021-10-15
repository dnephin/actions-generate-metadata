package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	actions "github.com/sethvargo/go-githubactions"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"strings"
)

const defaultRepositoryOwner string = "hashicorp"
const defaultMetadataFileName string = "metadata.json"

type input struct {
	branch           string
	filePath         string
	metadataFileName string
	product          string
	repo             string
	org              string
	sha              string
	version          string
}

type Metadata struct {
	Branch          string `json:"branch"`
	BuildWorkflowId string `json:"buildworkflowid"`
	Product         string `json:"product"`
	Repo            string `json:"repo""`
	Org             string `json:"org"`
	Revision        string `json:"sha"`
	Version         string `json:"version"`
}

func main() {
	if err := run(); err != nil {
		actions.Fatalf(err.Error())
	}
}

func run() error {
	in := input{
		branch:           actions.GetInput("branch"),
		filePath:         actions.GetInput("filePath"),
		metadataFileName: actions.GetInput("metadataFileName"),
		product:          actions.GetInput("product"),
		repo:             actions.GetInput("repo"),
		org:              actions.GetInput("org"),
		sha:              actions.GetInput("sha"),
		version:          actions.GetInput("version"),
	}
	generatedFile, err := createMetadataJson(in)
	if err != nil {
		return err
	}

	if err := checkFileIsExist(generatedFile); err != nil {
		return err
	}

	actions.SetOutput("filepath", generatedFile)
	actions.SetEnv("filepath", generatedFile)
	actions.Infof("Successfully created %v file\n", generatedFile)
	return nil
}

func checkFileIsExist(filepath string) error {
	fileInfo, err := os.Stat(filepath)

	if os.IsNotExist(err) {
		return err
	}
	if err != nil {
		return fmt.Errorf("failed to read file %v: %w", filepath, err)
	}
	// Return false if the fileInfo says the file path is a directory
	if !fileInfo.IsDir() {
		return fmt.Errorf("path is not a directory: %v", filepath)
	}
	return nil
}

func createMetadataJson(in input) (string, error) {
	branch := in.branch
	actions.Infof("GITHUB_HEAD_REF %v\n", os.Getenv("GITHUB_HEAD_REF"))
	actions.Infof("GITHUB_REF %v\n", os.Getenv("GITHUB_REF"))
	if branch == "" && os.Getenv("GITHUB_HEAD_REF") == "" {
		branch = strings.TrimPrefix(os.Getenv("GITHUB_REF"), "refs/heads/")
	} else {
		branch = os.Getenv("GITHUB_HEAD_REF")
	}

	actions.Infof("Working branch %v\n", branch)

	file := in.metadataFileName
	if file == "" {
		file = defaultMetadataFileName
	}
	filePath := path.Join(in.filePath, file)

	product := in.product
	if product == "" {
		return "", fmt.Errorf("Missing input 'product' value")
	}
	sha := in.sha
	if sha == "" {
		sha = os.Getenv("GITHUB_SHA")
	}
	actions.Infof("Working sha %v\n", sha)

	org := in.org
	if org == "" {
		org = defaultRepositoryOwner
	}
	repository := in.repo
	if repository == "" {
		repository = strings.Split(os.Getenv("GITHUB_REPOSITORY"), "/")[1]
	}

	runId := os.Getenv("GITHUB_RUN_ID")
	if runId == "" {
		return "", fmt.Errorf("GITHUB_RUN_ID is empty")
	}

	version := in.version
	if version == "" {
		return "", fmt.Errorf("The version or version command is not provided")
	} else if strings.Contains(version, " ") {
		var err error
		version, err = getVersion(version)
		if err != nil {
			return "", fmt.Errorf(err.Error())
		}
	}
	actions.Infof("Working version %v\n", version)

	actions.Infof("Creating metadata file in %v\n", filePath)

	m := &Metadata{
		Product:         product,
		Org:             org,
		Revision:        sha,
		BuildWorkflowId: runId,
		Version:         version,
		Branch:          branch,
		Repo:            repository}
	output, err := json.MarshalIndent(m, "", "\t\t")

	if err != nil {
		return "", fmt.Errorf("JSON marshal failure. Error:%v\n", output, err)
	} else {
		err = ioutil.WriteFile(filePath, output, 0644)
		if err != nil {
			return "", fmt.Errorf("Failed writing data into %v file. Error: %v\n", in.metadataFileName, err)
		}
	}
	return filePath, nil
}

func getVersion(command string) (string, error) {
	version, err := execCommand(strings.Fields(command)...)
	if err != nil {
		return "", err
	}
	if version == "" {
		return "", fmt.Errorf("Failed to setup version using %v command", command)
	}
	return strings.TrimSuffix(version, "\n"), nil
}

func execCommand(args ...string) (string, error) {
	name := args[0]
	stderr := new(bytes.Buffer)
	stdout := new(bytes.Buffer)

	cmd := exec.Command(name, args[1:]...)
	cmd.Stderr = stderr
	cmd.Stdout = stdout
	err := cmd.Run()
	actions.Infof("Running %v command: %v\nstdout: %v\nstderr: %v\n", name, cmd,
		strings.TrimSpace(string(stdout.Bytes())), strings.TrimSpace(string(stderr.Bytes())))

	if err != nil {
		return "", fmt.Errorf("Failed to run %v command %v: %v", name, cmd, err)
	}
	return string(stdout.Bytes()), nil
}
