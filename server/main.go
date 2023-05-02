package main

import (
	"bytes"
	"context"
	"encoding/csv"
	"fmt"
	"html/template"
	"log"
	"math"
	"os"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/rs/zerolog"
	"github.com/sashabaranov/go-openai"
	"github.com/spf13/cobra"
)

func serverCommand() *cobra.Command {
	var apiKey, logFile, fieldDataCSV, fieldDescriptionCSV string
	var cmd = &cobra.Command{
		Use:   "server",
		Short: "this is the backend for the chatGPT tool",
		Long:  ``,
		RunE: func(cmd *cobra.Command, args []string) error {
			server(apiKey, fieldDataCSV, logFile, fieldDescriptionCSV)
			return nil
		},
	}
	cmd.Flags().StringVarP(&apiKey, "apiKey", "k", "", "openAI api key")
	cmd.Flags().StringVarP(&logFile, "logfile", "l", "", "filepath to save logs")
	cmd.Flags().StringVarP(&fieldDataCSV, "fieldDataCSV", "i", "", "path to csv file with field annotations")
	cmd.Flags().StringVarP(&fieldDescriptionCSV, "fieldDescriptionCSV", "d", "", "path to csv file with field names and descriptions")
	return cmd
}

func main() {
	cmd := &cobra.Command{
		Use: "chatGPT-tool",
	}
	cmd.AddCommand(serverCommand())
	if err := cmd.Execute(); err != nil {
		os.Exit(0)
	}
}

type FieldData struct {
	ID          string   `json:"id"`
	Title       string   `json:"title"`
	Annotations []string `json:"annotations"`
}

type id struct {
	FieldID string `json:"id"`
}

type InputForGPT struct {
	Prompt      string `json:"prompt"`
	Clause      string `json:"clause"`
	Temperature string `json:"temp"`
	NumRuns     string `json:"numruns"`
}

type GPTOutput struct {
	Outputs []string `json:"gptoutputs"`
}

type AppState struct {
	Prompt      string   `json:"prompt"`
	Clause      string   `json:"clause"`
	Result      []string `json:"result"`
	Notes       string   `json:"notes"`
	Rating      string   `json:"rating"`
	Temperature string   `json:"temp"`
}

type docAnnotation struct {
	docid, text string
}

func convertToSlice(docs map[string]string) []string {
	da := make([]string, 0, len(docs))
	for _, v := range docs {
		da = append(da, v)
	}
	return da
}

// readAnnotationCSV reads all of the field annotations from the csv
func readAnnotationCSV(path string) (map[string][]string, error) {
	// read csv file with field annotations
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	csvReader := csv.NewReader(f)
	records, err := csvReader.ReadAll()
	if err != nil {
		return nil, err
	}
	annotationMap := make(map[string][]string)
	fieldAnnotations := map[string]string{}
	for i := 0; i < len(records)-1; i++ {
		fieldID := records[i][0]
		docID := records[i][1]
		text := records[i][2]
		if text == "" { // skip blank annotations
			continue
		}
		if _, ok := fieldAnnotations[docID]; !ok {
			fieldAnnotations[docID] = text
		} else {
			fieldAnnotations[docID] += "\n" + text
		}
		//fieldAnnotations = append(fieldAnnotations, text)
		nextFieldID := records[i+1][0]
		// if the next row starts a new field, save the current field
		// and clear the fieldAnnotations slice
		if nextFieldID != fieldID {
			if err != nil {
				return nil, err
			}
			annotationMap[fieldID] = convertToSlice(fieldAnnotations)
			fieldAnnotations = map[string]string{}
		}
	}
	// finish the last field because the loop above won't get it
	fieldID := records[len(records)-1][0]
	docID := records[len(records)-1][1]
	text := records[len(records)-1][2]
	if _, ok := fieldAnnotations[docID]; !ok {
		fieldAnnotations[docID] = text
	} else {
		fieldAnnotations[docID] += "\n" + text
	}
	//fieldAnnotations = append(fieldAnnotations, text)
	annotationMap[fieldID] = convertToSlice(fieldAnnotations)
	if err != nil {
		return nil, err
	}
	return annotationMap, nil
}

// readFieldDescriptionsCSV creates a map of field id to field name from a csv file
func readFieldDescriptionsCSV(path string) (map[string]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	csvReader := csv.NewReader(f)
	records, err := csvReader.ReadAll()
	if err != nil {
		return nil, err
	}
	fieldInfoMap := make(map[string]string, len(records))
	for i := range records[1:] {
		record := records[i+1]
		fieldInfoMap[record[0]] = record[1]
	}
	return fieldInfoMap, nil
}

type PromptClause struct {
	Clause string
}

func submitRequest(inputString, apiKey string, temp float64, n int) ([]string, error) {
	client := openai.NewClient(apiKey)
	resp, err := client.CreateChatCompletion(
		context.Background(),
		openai.ChatCompletionRequest{
			Model:       openai.GPT3Dot5Turbo,
			Temperature: float32(temp),
			N:           n,
			Messages: []openai.ChatCompletionMessage{
				{
					Role:    openai.ChatMessageRoleSystem,
					Content: "You are a helpful legal assistant designed to respond to legal questions and prompts. Answer as concisely as possible.",
				},
				{
					Role:    openai.ChatMessageRoleUser,
					Content: inputString,
				},
			},
		},
	)
	if err != nil {
		return nil, err
	}
	// get the response choices
	res := make([]string, len(resp.Choices))
	for i, v := range resp.Choices {
		res[i] = strings.ReplaceAll(v.Message.Content, "\n", "</br>")
	}
	return res, nil
}

func server(apiKey, fieldDataCSV, logFile, fieldDescriptionCSV string) {
	app := fiber.New()
	app.Use(cors.New(cors.Config{
		AllowOrigins: "http://localhost:3000",
		AllowHeaders: "Origin, Content-Type, Accept",
	}))

	// read field data
	fmt.Println("Reading Annotation CSV...")
	allFields, err := readAnnotationCSV(fieldDataCSV)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println("...Done")

	fmt.Println("Reading Field Descriptions")
	// create map of field ids to field titles
	idToNameMap, err := readFieldDescriptionsCSV(fieldDescriptionCSV)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println("...Done")
	// create logger
	f, err := os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0664)
	if err != nil {
		fmt.Println(err)
	}
	defer f.Close()
	logger := zerolog.New(f).With().Timestamp().Logger()

	// when "Get Field" button is clicked
	app.Post("/api/getField", func(c *fiber.Ctx) error {
		res := id{} // get field id
		if err := c.BodyParser(&res); err != nil {
			fmt.Println(err)
		}
		fmt.Println("Getting annotations for field...", res.FieldID)
		annotations, ok := allFields[res.FieldID]
		if !ok {
			fmt.Println("...field id not found in annotation csv")
			return c.JSON(FieldData{ID: res.FieldID, Title: "field not found!"})
		}
		// field's annotations
		title, ok := idToNameMap[res.FieldID]
		if !ok {
			fmt.Println("...field id not found in field description csv")
			return c.JSON(FieldData{ID: res.FieldID, Title: "field not found!"})
		}
		fmt.Println("...success")
		// return corresponding field
		return c.JSON(FieldData{ID: res.FieldID, Title: title, Annotations: annotations})
	})

	// When "Run" button is clicked
	app.Post("/api/run", func(c *fiber.Ctx) error {
		gptInput := InputForGPT{}
		if err := c.BodyParser(&gptInput); err != nil {
			fmt.Println(err)
		}
		var inputString string
		// create the input string for GPT
		tmpl, err := template.New("gpt").Parse(gptInput.Prompt)
		if err != nil || !strings.Contains(gptInput.Prompt, "{{.Clause}}") {
			fmt.Printf("Defaulting to base template: %w\n", err)
			inputString = gptInput.Prompt + "\n" + gptInput.Clause
		} else {
			var buf bytes.Buffer
			tmpl.Execute(&buf, PromptClause{gptInput.Clause})
			inputString = buf.String()
		}
		// parse and round float
		temperature, err := strconv.ParseFloat(gptInput.Temperature, 32)
		temperature = math.Round(temperature*10) / 10
		if err != nil {
			fmt.Printf("Error converting temperature string to float: %w\n", err)
		}
		// parse numRuns int
		numRuns, err := strconv.Atoi(gptInput.NumRuns)
		if err != nil {
			fmt.Printf("Error converting numRuns string to int: %w\n", err)
		}
		// get the outputs from GPT
		gptOutputs, err := submitRequest(inputString, apiKey, temperature, numRuns)
		if err != nil {
			fmt.Printf("Error in submitRequest from OpenAI: %w\n", err)

		}
		return c.JSON(GPTOutput{Outputs: gptOutputs})
	})

	// when "Save" or "Save and Continue" buttons are clicked log the app's state
	app.Post("/api/save", func(c *fiber.Ctx) error {
		res := AppState{} // get field id
		if err := c.BodyParser(&res); err != nil {
			fmt.Println(err)
		}
		// log the current state
		logger.Info().Str("prompt", res.Prompt).Str("clause", res.Clause).Strs("result", res.Result).Str("rating", res.Rating).Str("notes", res.Notes).Str("temperature", res.Temperature).Msg("Info message")
		return c.JSON(fiber.Map{"status": "success"})
	})
	log.Fatal(app.Listen(":4000"))

}
