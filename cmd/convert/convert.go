package main

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

func main() {

	if len(os.Args) < 3 {
		fmt.Println("Usage: convert <dataset path> <output path> ")
		return
	}

	dataSetDir := os.Args[1]
	outputPath := os.Args[2]
	metadataFileName := fmt.Sprintf("%s/%s", dataSetDir, "all_sources_metadata_2020-03-13.csv")
	metadataFile, err := os.Open(metadataFileName)
	if err != nil {
		fmt.Sprintf("Error opening file %s: %s \n", metadataFileName, err)
		return
	}

	reader := csv.NewReader(metadataFile)

	//Skip header
	reader.Read()
	record, err := reader.Read()
	for err == nil {
		processFile(dataSetDir, outputPath, record)
		record, err = reader.Read()
	}

}

func processFile(dataSetDir string, outputDir string, metadata []string) {

	fileName := metadata[0]
	hasFullText := metadata[len(metadata)-1]

	if hasFullText == "True" {
		filePath, err := findFile(dataSetDir, fileName+".json")
		if err != nil {
			fmt.Printf("Warn: error opening file %s: %s \n", fileName, err)
			return
		}
		fileContent, err := ioutil.ReadFile(filePath)
		if err != nil {
			fmt.Printf("Warn: error reading file %s: %s \n", filePath, err)
			return
		}
		var content interface{}
		err = json.Unmarshal(fileContent, &content)
		if err != nil {
			fmt.Printf("Warn: error parsing file %s: %s \n", filePath, err)
			return
		}

		jsonConent := content.(map[string]interface{})
		textContent, err := convertContent(jsonConent)
		if err != nil {
			fmt.Printf("Warn: error converting file %s: %s\n", filePath, err)
			return
		}
		outputFile := outputDir + "/" + fileName + ".txt"
		err = ioutil.WriteFile(outputFile, textContent, 0644)
		if err != nil {
			fmt.Printf("Warn: error writing file %s: %s\n", outputFile, err)
			return
		}

	}
}

func convertContent(jsonContent map[string]interface{}) ([]byte, error) {

	var result strings.Builder
	metadata := jsonContent["metadata"].(map[string]interface{})
	title := metadata["title"].(string)
	result.WriteString("\nTITLE\n" + title + "\n")

	result.WriteString("\nAUTHORS\n")

	authors := metadata["authors"].([]interface{})

	result.WriteString(formatAuthors(authors))

	abstract := jsonContent["abstract"].([]interface{})

	abstractText, err := convertToText(abstract)
	if err != nil {
		return nil, err
	}
	result.WriteString("\n" + abstractText)

	body := jsonContent["body_text"].([]interface{})
	bodyText, err := convertToText(body)
	if err != nil {
		return nil, err
	}
	result.WriteString(bodyText)

	figReferences, hasFigReferences := jsonContent["ref_entries"].(map[string]interface{})

	if hasFigReferences {
		formattedFigReferences := formatFigReferences(figReferences)
		if formattedFigReferences != "" {
			result.WriteString("\nFIGURES AND TABLES\n\n")
			result.WriteString(formattedFigReferences)
		}

	}
	references, hasReferences := jsonContent["bib_entries"].(map[string]interface{})

	if hasReferences {
		formattedReferences := formatReferences(references)
		if formattedReferences != "" {
			result.WriteString("\nREFERENCES\n\n")
			result.WriteString(formattedReferences)

		}

	}

	return []byte(result.String()), nil

}

func formatFigReferences(figReferences map[string]interface{}) string {
	var result strings.Builder

	for refID, r := range figReferences {
		reference := r.(map[string]interface{})
		result.WriteString(refID)
		result.WriteString("\n")
		text := reference["text"].(string)
		result.WriteString(text)
		result.WriteString("\n")

	}

	return result.String()

}
func formatReferences(references map[string]interface{}) string {

	var result strings.Builder

	for _, r := range references {
		reference := r.(map[string]interface{})
		refID := reference["ref_id"].(string)
		title := reference["title"].(string)
		authors := reference["authors"].([]interface{})
		venue, hasVenue := reference["venue"].(string)
		hasVenue = hasVenue && venue != ""
		year, hasYear := (reference["year"].(float64))
		result.WriteString(refID + ". ")
		result.WriteString(formatAuthorReferences(authors))
		result.WriteString("\n")
		result.WriteString(title)
		result.WriteString("\n")
		if hasVenue {
			result.WriteString(venue)
		}
		if hasYear {
			if hasVenue {
				result.WriteString(" - ")
			}
			result.WriteString(fmt.Sprintf("%.0f\n", year))
		}
		result.WriteString("\n")

	}

	return result.String()
}

func formatAuthorReferences(authors []interface{}) string {
	var result strings.Builder
	first := true
	for _, a := range authors {
		author := a.(map[string]interface{})
		if !first {
			result.WriteString(" - ")
		} else {
			first = false
		}

		result.WriteString(author["first"].(string))
		result.WriteString(" ")
		result.WriteString(author["last"].(string))
	}

	return result.String()
}

func formatAuthors(authors []interface{}) string {

	var result strings.Builder
	for _, a := range authors {
		author := a.(map[string]interface{})
		result.WriteString(author["first"].(string))
		result.WriteString(" " + author["last"].(string))
		affiliation, hasAffiliation := author["affiliation"].(map[string]interface{})
		if hasAffiliation {
			result.WriteString(" - ")
			institution, hasInstitution := affiliation["institution"].(string)
			if hasInstitution {
				result.WriteString(institution)
			}
			location, hasLocation := affiliation["location"].(map[string]interface{})
			if hasLocation {
				country, hasCountry := location["country"].(string)
				if hasCountry {
					result.WriteString(" - ")
					result.WriteString(country)
				}
			}

		}
		result.WriteString("\n")

	}

	return result.String()

}
func convertToText(content []interface{}) (string, error) {
	var result strings.Builder

	currentSection := ""

	for _, p := range content {
		paragraph := p.(map[string]interface{})
		section := paragraph["section"].(string)
		if currentSection != section {
			result.WriteString("\n" + section + "\n\n")
			currentSection = section
		}
		text := paragraph["text"].(string)
		result.WriteString(text + "\n")
	}

	return result.String(), nil

}
func findFile(rootDir string, fileName string) (string, error) {

	result := ""
	filepath.Walk(rootDir, func(path string, info os.FileInfo, err error) error {

		if strings.HasSuffix(path, fileName) {
			result = path
		}
		return nil
	})

	if result == "" {
		return result, fmt.Errorf("file not found: %s ", fileName)
	}
	return result, nil
}
