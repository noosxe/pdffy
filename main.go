package main

import (
	"encoding/xml"
	"flag"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"unicode"
)

type Question struct {
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

func main() {
	fmt.Print("pdffy ======== not sure why?\n\n")

	fileFlag := flag.String("f", "", "input xml file")
	flag.Parse()

	if len(*fileFlag) == 0 {
		fmt.Println("please provide the input file path")
		os.Exit(1)
	}

	fileArg := *fileFlag
	fmt.Printf("trying to open %s\n", fileArg)
	xmlFile, err := os.Open(fileArg)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	defer xmlFile.Close()

	byteValue, err := io.ReadAll(xmlFile)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	var xmlDoc XmlDoc
	err = xml.Unmarshal(byteValue, &xmlDoc)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	questions := make([]Question, 0)

	for _, page := range xmlDoc.Pages {
		fmt.Printf("\npage: %d\n", page.Number)
		fmt.Printf("images: %d\n", len(page.Images))
		fmt.Printf("texts: %d\n", len(page.Texts))

		// var question Question
		var sb strings.Builder
		var i int
		var question Question
		waitingForQ := true
		for i < len(page.Texts) {
			text := page.Texts[i]
			clean := strings.TrimSpace(text.Content)
			if len(clean) == 0 {
				i += 1
				continue
			}
			fmt.Printf("-- %s\n", clean)

			runes := []rune(clean)

			first := runes[0]

			if unicode.IsDigit(first) {
				fmt.Printf("found digit: %v\n", first)
				second := runes[1]

				if second == '.' {
					if waitingForQ {
						q := sb.String()
						sb.Reset()
						question.Text = q
						fmt.Printf("acc: %s\n", q)

						waitingForQ = false
						fmt.Println("not waiting for Q")

						sb.WriteString(clean)
					} else {
						question.Options = append(question.Options, sb.String())
						sb.Reset()
						sb.WriteString(clean)
					}
				}
			} else {
				if clean == "Պատ" {
					if sb.Len() > 0 {
						question.Options = append(question.Options, sb.String())
						sb.Reset()
					}

					ans := page.Texts[i+2]
					ansRunes := []rune(ans.Content)

					if ansRunes[0] != '՝' {
						fmt.Printf("expected the tick but got: %v", ansRunes[0])
						os.Exit(1)
					}

					ansNum, err := strconv.Atoi(string(ansRunes[1]))
					if err != nil {
						fmt.Printf("deceptive number i see: %s\n", ans.Content)
						os.Exit(1)
					}

					question.Answer = ansNum
					questions = append(questions, question)
					question = Question{}
					waitingForQ = true
					fmt.Println("waiting for Q")

					i += 3
					continue
				}

				sb.WriteRune(' ')
				sb.WriteString(clean)
			}

			i++

		}
	}

	fmt.Println(questions)
}
