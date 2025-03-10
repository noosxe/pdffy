package main

import (
	"encoding/json"
	"encoding/xml"
	"flag"
	"fmt"
	"io"
	"math"
	"os"
	"strconv"
	"strings"
	"unicode"

	"github.com/noosxe/pdffy/pkg/stm"
)

type Questionnaire struct {
	Questions []Question `json:"questions"`
}

type Question struct {
	Id      string   `json:"id"`
	Text    string   `json:"text"`
	Image   *string  `json:"image,omitempty"`
	Options []string `json:"options"`
	Answer  int      `json:"answer"`
}

type XmlDoc struct {
	Pages []Page `xml:"page"`
}

type Page struct {
	Number int     `xml:"number,attr"`
	Images []Image `xml:"image"`
	Texts  []Text  `xml:"text"`
}

type Image struct {
	Top    int    `xml:"top,attr"`
	Left   int    `xml:"left,attr"`
	Width  int    `xml:"width,attr"`
	Height int    `xml:"height,attr"`
	Src    string `xml:"src,attr"`
}

type Text struct {
	Top     int    `xml:"top,attr"`
	Left    int    `xml:"left,attr"`
	Width   int    `xml:"width,attr"`
	Height  int    `xml:"height,attr"`
	Content string `xml:",chardata"`
}

type Rect struct {
	Top    int
	Bottom int
	Left   int
	Right  int
}

type MultiFile []string

func (mf *MultiFile) Set(value string) error {
	*mf = append(*mf, value)
	return nil
}

func (mf *MultiFile) String() string {
	return strings.Join(*mf, ", ")
}

func main() {
	fmt.Print("pdffy ======== not sure why?\n\n")

	var files MultiFile
	flag.Var(&files, "file", "input xml file")
	outFlag := flag.String("o", "", "output json file")
	flag.Parse()

	fmt.Println(files)

	if len(files) == 0 {
		fmt.Println("please provide the input file path")
		os.Exit(1)
	}

	allQuestions := make([]Question, 0)
	for i, file := range files {
		questions, err := ProcessFile(file, fmt.Sprintf("Book %d", i+1))
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		allQuestions = append(allQuestions, questions...)
	}

	if *outFlag != "" {
		fmt.Printf("writing output to: %s\n", *outFlag)
		questionnaire := Questionnaire{Questions: allQuestions}
		jstring, err := json.MarshalIndent(questionnaire, "", "  ")
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		f, err := os.Create(*outFlag)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		defer f.Close()

		_, err = f.Write(jstring)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	} else {
		fmt.Println(allQuestions)
	}
}

func ProcessFile(filePath string, idPrefix string) ([]Question, error) {
	fmt.Printf("trying to open %s\n", filePath)
	xmlFile, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer xmlFile.Close()

	byteValue, err := io.ReadAll(xmlFile)
	if err != nil {
		return nil, err
	}

	var xmlDoc XmlDoc
	err = xml.Unmarshal(byteValue, &xmlDoc)
	if err != nil {
		return nil, err
	}

	questions := make([]Question, 0)
	for i, page := range xmlDoc.Pages {
		texts := SanitizeTexts(page.Texts)

		var question Question
		var sb strings.Builder
		var rect Rect
		var iQ int
		machine := stm.StateMachine[Text]{}
		machine.Init(texts).
			AddState(stm.State[Text]{
				First: true,
				Name:  "first",
				Run: func(value Text, stm *stm.StateMachine[Text]) error {
					if sb.Len() > 0 {
						sb.WriteRune(' ')
					}

					sb.WriteString(value.Content)
					fmt.Println(value.Content)

					next, err := stm.Token(1)
					if err != nil {
						return err
					}
					rect = ExpandRect(rect, value)

					if IsOption(next.Value) {
						q := sb.String()
						sb.Reset()
						question.Text = q
						return stm.Next("option")
					}

					return stm.Next("first")
				},
			}).
			AddState(stm.State[Text]{
				Name: "option",
				Run: func(value Text, stm *stm.StateMachine[Text]) error {
					fmt.Println(value.Content)

					next, err := stm.Token(1)
					if err != nil {
						return err
					}

					if IsOption(value) {
						if sb.Len() > 0 {
							question.Options = append(question.Options, sb.String()[2:])
							sb.Reset()
						}
					}
					rect = ExpandRect(rect, value)

					if sb.Len() > 0 {
						sb.WriteRune(' ')
					}

					sb.WriteString(value.Content)

					if IsAnswer(next.Value) {
						if sb.Len() > 0 {
							question.Options = append(question.Options, sb.String()[2:])
							sb.Reset()
						}

						return stm.Next("answer")
					}

					return stm.Next("option")
				},
			}).
			AddState(stm.State[Text]{
				Name: "answer",
				Run: func(value Text, stm *stm.StateMachine[Text]) error {
					next, err := stm.Token(1)
					if err != nil {
						return err
					}

					runes := []rune(next.Value.Content)
					if runes[0] != '․' {
						return fmt.Errorf("unexpected token in answer section: %s", next.Value.Content)
					}
					stm.Consume(1)

					next, err = stm.Token(1)
					if err != nil {
						return err
					}

					runes = []rune(next.Value.Content)
					if runes[0] != '՝' {
						return fmt.Errorf("unexpected token in answer section: %s", next.Value.Content)
					}
					stm.Consume(1)

					fmt.Println(string(runes[1]))
					ansNum, err := strconv.Atoi(string(runes[1]))
					if err != nil {
						return err
					}
					question.Answer = ansNum - 1
					for _, img := range page.Images {
						if IsInRect(rect, img) {
							question.Image = &img.Src
						}
					}
					question.Id = fmt.Sprintf("%s, Page %d, Question %d", idPrefix, i+1, iQ+1)

					questions = append(questions, question)
					question = Question{}
					rect = Rect{}
					iQ++
					return stm.Next("first")
				},
			})

		err = machine.Parse()
		if err != nil {
			return nil, err
		}
	}

	return questions, nil
}

func IsOption(text Text) bool {
	runes := []rune(text.Content)
	first := runes[0]
	if !unicode.IsDigit(first) {
		return false
	}

	return true
	// second := runes[1]
	// return second == '.'
}

func IsAnswer(text Text) bool {
	return text.Content == "Պատ"
}

func ExpandRect(rect Rect, text Text) Rect {
	if rect.Top == 0 || text.Top < rect.Top {
		rect.Top = text.Top
	}

	if rect.Left == 0 || text.Left < rect.Left {
		rect.Left = text.Left
	}

	rect.Bottom = int(math.Max(float64(rect.Bottom), float64(text.Top)+float64(text.Height)))
	rect.Right = int(math.Max(float64(rect.Right), float64(text.Left)+float64(text.Width)))

	return rect
}

func SanitizeTexts(texts []Text) []Text {
	var result []Text

	for _, text := range texts {
		text.Content = strings.TrimSpace(text.Content)
		if text.Content != "" {
			result = append(result, text)
		}
	}

	return result
}

func IsInRect(rect Rect, image Image) bool {
	return image.Left > rect.Left && image.Top > rect.Top && (image.Left+image.Width) < rect.Right && (image.Top+image.Height) < rect.Bottom
}
