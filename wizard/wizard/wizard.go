package wizard

import (
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"strconv"
	"strings"

	"gopkg.in/yaml.v1"

	"github.com/mgutz/ansi"
	"github.com/phrase/phraseapp-go/phraseapp"
)

type WizardData struct {
	AccessToken string `yaml:"access_token"`
	ProjectId   string `yaml:"project_id"`
	Format      string `yaml:"file_format"`
	Step        string `yaml:"-"`
}

type WizardWrapper struct {
	Data *WizardData `yaml:"phraseapp"`
}

func clean() {
	switch runtime.GOOS {
	case "darwin":
		cmd := exec.Command("clear")
		cmd.Stdout = os.Stdout
		cmd.Run()
	case "linux":
		cmd := exec.Command("clear")
		cmd.Stdout = os.Stdout
		cmd.Run()
	case "windows":
		cmd := exec.Command("cls")
		cmd.Stdout = os.Stdout
		cmd.Run()
	default:
		fmt.Printf("%s unsupported", runtime.GOOS)
		panic("Do not know")
	}
}

func printError(errorMsg string) {
	red := ansi.ColorCode("red+b:black")
	reset := ansi.ColorCode("reset")

	fmt.Println(red, errorMsg, reset)
}

func printSuccess(msg string) {
	green := ansi.ColorCode("green+b:black")
	reset := ansi.ColorCode("reset")

	fmt.Println(green, msg, reset)
}

func DisplayWizard(data *WizardData, step string, errorMsg string) {
	clean()

	if errorMsg != "" {
		printError(errorMsg)
	}
	switch {

	case step == "" || data.AccessToken == "":
		data.Step = "token"
		tokenStep(data)
		return
	case step == "newProject":
		data.Step = "newProject"
		newProjectStep(data)
		return
	case step == "selectProject":
		data.Step = "selectProject"
		selectProjectStep(data)
		return
	case step == "selectFormat":
		data.Step = "selectFormat"
		selectFormat(data)
		return
	case step == "finish":
		writeConfig(data, ".phraseapp.yaml")
		return
	}

}

func selectFormat(data *WizardData) {
	auth := phraseapp.AuthCredentials{Token: data.AccessToken}
	phraseapp.RegisterAuthCredentials(&auth, nil)
	formats, err := phraseapp.FormatsList(1, 25)
	if err != nil {
		panic(err.Error())
	}

	for counter, format := range formats {
		fmt.Printf("%2d. %s - %s, File-Extension: %s\n", counter+1, format.ApiName, format.Name, format.Extension)
	}

	var id string
	fmt.Printf("Select the format you want to use for language files you download from PhraseApp (e.g. enter 1 for %s): ", formats[0].Name)
	fmt.Scanln(&id)
	number, err := strconv.Atoi(id)
	if err != nil || number < 1 || number > len(formats)+1 {
		DisplayWizard(data, "selectFormat", fmt.Sprintf("Argument Error: Please select a format from the list by specifying its position in the list."))
		return
	}
	data.Format = formats[number-1].ApiName
	DisplayWizard(data, next(data), "")
}

func writeConfig(data *WizardData, filename string) {
	wrapper := WizardWrapper{Data: data}
	bytes, err := yaml.Marshal(wrapper)
	if err != nil {
		panic(err.Error())
	}
	str := fmt.Sprintf("Success! We have created the config file for you %s:", filename)
	printSuccess(str)
	fmt.Println("")
	fmt.Println(string(bytes))

	printSuccess("You can make changes to this file, see this documentation for more advanced options: http://docs.phraseapp.com/api/v2/config")
	printSuccess("Now start using phraseapp push & pull for your workflow:")
	fmt.Println("")
	fmt.Println("$ phraseapp push")
	fmt.Println("$ phraseapp pull")
	fmt.Println("")
	var initialPush string
	fmt.Print("Enter yes to push your locales now for the first time: ")
	fmt.Scanln(&initialPush)
	if initialPush == "y" {
		fmt.Println("Pushing....")
	}
	fmt.Println("Setup completed!")
}

func next(data *WizardData) string {
	switch data.Step {
	case "", "token":
		return "selectProject"
	case "selectProject":
		return "selectFormat"
	case "newProject":
		return "selectFormat"
	case "selectFormat":
		return "finish"
	}
	return ""
}

func tokenStep(data *WizardData) {
	fmt.Print("Please enter you API Access Token (Generate one in your profile at phraseapp.com): ")
	fmt.Scanln(&data.AccessToken)
	data.AccessToken = strings.ToLower(data.AccessToken)
	success, err := regexp.MatchString("^[0-9a-f]{64}$", data.AccessToken)
	if err != nil {
		panic(err.Error())
	}
	if success == true {
		DisplayWizard(data, next(data), "")
	} else {
		data.AccessToken = ""
		DisplayWizard(data, "", "Argument Error: AccessToken must be 64 letters long and can only contain a-f, 0-9")
	}
}

func newProjectStep(data *WizardData) {
	fmt.Print("Enter name of new project: ")
	projectParam := &phraseapp.ProjectParams{}
	fmt.Scanln(&projectParam.Name)

	res, err := phraseapp.ProjectCreate(projectParam)
	if err != nil {
		success, match_err := regexp.MatchString("401", err.Error())
		if match_err != nil {
			fmt.Println(err.Error())
			panic(match_err.Error())
		}
		if success {
			data.AccessToken = ""
			DisplayWizard(data, "", fmt.Sprintf("Argument Error: Your AccessToken '%s' has no write scope. Please create a new Access Token with read and write scope.", data.AccessToken))
		} else {
			success, match_err := regexp.MatchString("Validation failed", err.Error())
			if match_err != nil {
				fmt.Println(err.Error())
				panic(match_err.Error())
			}
			if success {
				DisplayWizard(data, "newProject", err.Error())
				return
			} else {
				panic(err.Error())
			}
		}
	} else {
		data.ProjectId = res.Id
		DisplayWizard(data, next(data), "")
		return
	}
}

func selectProjectStep(data *WizardData) {
	auth := phraseapp.AuthCredentials{Token: data.AccessToken}
	fmt.Println("Please select your project:")
	phraseapp.RegisterAuthCredentials(&auth, nil)
	projects, err := phraseapp.ProjectsList(1, 25)
	if err != nil {
		success, match_err := regexp.MatchString("401", err.Error())
		if match_err != nil {
			fmt.Println(err.Error())
			panic(match_err.Error())
		}
		if success {
			errorMsg := fmt.Sprintf("Argument Error: AccessToken '%s' is invalid. It may be revoked. Please create a new Access Token.", data.AccessToken)
			data.AccessToken = ""
			DisplayWizard(data, "", errorMsg)
		} else {
			panic(err.Error())
		}
	}

	if len(projects) == 1 {
		data.ProjectId = projects[0].Id
		fmt.Printf("You've got one project, %s. Answer \"y\" to select this or \"n\" to create a new project: ")
		var answer string
		fmt.Scanln(&answer)
		if answer == "y" {
			DisplayWizard(data, next(data), "")
			return
		} else {
			data.ProjectId = ""
			DisplayWizard(data, "newProject", "")
			return
		}
	}
	for counter, project := range projects {
		fmt.Printf("%2d. %s (Id: %s)\n", counter+1, project.Name, project.Id)
	}
	fmt.Printf("%2d. Create new project\n", len(projects)+1)
	fmt.Print("Select project: ")
	var id string
	fmt.Scanln(&id)
	number, err := strconv.Atoi(id)
	if err != nil || number < 1 || number > len(projects)+1 {
		DisplayWizard(data, "selectProject", fmt.Sprintf("Argument Error: Please select a project from the list by specifying its position in the list, e.g. 2 for the second project."))
		return
	}

	if number == len(projects)+1 {
		DisplayWizard(data, "newProject", "")
		return
	}

	selectedProject := projects[number-1]
	data.ProjectId = selectedProject.Id
	DisplayWizard(data, next(data), "")
}
