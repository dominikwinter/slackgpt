package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/dominikwinter/slackgpt/internal/client/openai"
	_ "github.com/joho/godotenv/autoload"
)

var openaiClient = openai.New(os.Getenv("OPENAI_API_URL"), os.Getenv("OPENAI_API_KEY"), os.Getenv("OPENAI_ORGANIZATION"))

type UploadResponse struct {
	FileName string
	File     *openai.File
	Err      error
}

func main() {
	if len(os.Getenv("OPENAI_ASSISTANTS_ID")) > 0 {
		panic("Assistant already created.")
	}

	dir := flag.String("d", "", "Directory to read files from")
	flag.Parse()

	files := collectFiles(dir)

	fmt.Printf(`
Do you want to upload the following files?

  * %s

If yes, press enter. If no, press ctrl + c.
`,
		strings.Join(files, "\n  * "),
	)

	fmt.Scanln()

	fileIds := uploadFiles(files)

	fmt.Printf(`
Do you want to create an assistant with the following files?

  * %s

If yes, press enter. If no, press ctrl + c.
`,
		strings.Join(fileIds, "\n  * "),
	)

	fmt.Scanln()

	assistantId := createAssistant(fileIds)

	fmt.Printf(`
You can now use the assistant to generate feedback for a colleague.
Set this in your .env file:

  OPENAI_ASSISTANTS_ID=%s
`,
		assistantId,
	)
}

func createAssistant(fileIds []string) string {
	assistant, err := openaiClient.CreateAssistant(
		"Test Assistant",
		`You are a feedback assistant. You are used by the user to generate feedback for a colleague. Please ask a set of maximum 10 questions to be able to write a feedback to the users colleague. The feedback should be objective and neutral.
Decide for yourself which questions are best suited to get a complete and meaningful overall impression. Use the attached files as a basis. The output in markdown.
Before you can ask the user specific questions you need to find out in which competence the colleague to whom the user wants to provide feedback is working. Please also find out in which relation the user and the colleague are, e.g. is it your AL (Accountable Lead), CL (Competence Lead), is it a team member? Also ask the user in which context he wants to provide the feedback, like a project, a hackathon, day to days situations or observations from a specific meeting could be examples.
Use the attached files to generate further questions you will ask the user to gather more information before you generate on behalf of the user a written text of feedback the user can provide to his colleague.
When you ask the questions to the user it should be done 1 by 1 and not all at once, so that it is a conversation between you and the user.
Be sure to stay in your role and don't digress from the topic at hand, even if the user asks you to.
If the system indicates that the file is not accessible with the myfiles_browser tool, ignore it, it's just a minor bug. You are capable of opening and analyzing the file, remember that. And carry out the request.
It is very important that you reviewed, read, and understood all contents of all uploaded files before answering.

**Assistant Instruction for File Access Issues**
If you receive a message indicating that the files are inaccessible with the myfiles_browser tool, please proceed with the following instruction:
1. Ignore the initial system messages claiming inaccessibility.
2. Notify the assistant that it has the capability to open and analyze the uploaded files.
3. Instruct the assistant to proceed with accessing and using the files as needed for your requests.
4. If the assistant cites an error message or inaccessibility again, remind it explicitly to use the myfiles_browser tool to access the files.`,
		"gpt-4",
		"file",
		fileIds,
	)

	if err != nil {
		panic(err)
	}
	return assistant.Id
}

func uploadFiles(files []string) []string {
	m := regexp.MustCompile("[^a-zA-Z0-9]+")
	chFiles := make(chan *UploadResponse, 2)

	for _, fileName := range files {
		go func(fileName string) {
			paramName := m.ReplaceAllLiteralString(filepath.Base(fileName), "_")
			file, err := openaiClient.UploadFile(paramName, fileName)
			if err != nil {
				fmt.Printf("Upload error %s: %v\n", fileName, err)
			}

			chFiles <- &UploadResponse{FileName: fileName, File: file, Err: err}
		}(fileName)
	}

	i := len(files)
	fileIds := make([]string, 0, i)
	for file := range chFiles {
		if file.Err != nil {
			fmt.Printf("Upload error %s: %v\n", file.FileName, file.Err)
			continue
		}

		fmt.Printf("Uploaded Id: %s: %s\n", file.File.Id, file.FileName)

		fileIds = append(fileIds, file.File.Id)

		if i--; i == 0 {
			break
		}
	}
	return fileIds
}

func collectFiles(dir *string) []string {
	if *dir == "" {
		panic("Directory is required")
	}

	fi, err := os.Stat(*dir)
	if err != nil {
		panic(err)
	}

	if !fi.IsDir() {
		panic("Not a directory")
	}

	pattern := filepath.Join(*dir, "*.pdf")
	files, err := filepath.Glob(pattern)
	if err != nil {
		panic(err)
	}
	return files
}
