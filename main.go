package main

import (
	"errors"
	"fmt"
	"gopkg.in/yaml.v3"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strconv"
)

const (
	WORKSPACE = "WORKSPACE"
)

func isDir(path string) bool {
	fileInfo, error := os.Stat(path)
	return error == nil && fileInfo.IsDir()
}

func readYmlToMap(ymlPath string) (map[string]interface{}, error) {
	f, err := os.Open(ymlPath)
	if err != nil {
		return nil, err
	}

	defer func() {
		if f != nil {
			f.Close()
		}
	}()
	decode := yaml.NewDecoder(f)
	ymlMap := map[string]interface{}{}
	err = decode.Decode(ymlMap)
	if err != nil {
		return nil, err
	}
	return ymlMap, nil
}

func arrayNoEmpty(array []string) []string {
	if array == nil {
		return []string{}
	}

	if len(array) == 0 {
		return array
	}

	newArray := []string{}
	for _, element := range array {
		if element != "" {
			newArray = append(newArray, element)
		}
	}
	return newArray
}

func getMapValueByPath(mapPath string, someMap interface{}) interface{} {
	if someMap != nil {
		compile := regexp.MustCompile(`[\.\[\]]`)
		split := arrayNoEmpty(compile.Split(mapPath, -1))
		if split != nil && len(split) != 0 {
			eachObj := someMap
			for _, element := range split {
				typeName := reflect.TypeOf(eachObj).String()
				if typeName == "map[string]interface {}" {
					eachObj = eachObj.(map[string]interface{})[element]
				} else if typeName == "[]interface {}" {
					elementIndex, error := strconv.ParseInt(element, 10, 64)
					if error != nil {
						return nil
					}
					eachObj = eachObj.([]interface{})[elementIndex]
				} else {
					return nil
				}

				if eachObj == nil {
					return nil
				}

			}
			return eachObj
		}
	}
	return someMap
}

func getProjRelativePath(ymlMap map[string]interface{}) (string, bool) {
	result := getMapValueByPath("projRelativePath", ymlMap)
	if result == nil {
		fmt.Println("yml須提參數:projRelativePath")
		return "", false
	} else {
		resultString, ok := result.(string)
		if !ok {
			fmt.Println("參數:projRelativePath 非字串")
			return "", false
		}
		fmt.Println("參數:projRelativePath:", resultString)
		return resultString, true
	}
}

func getExcludeRegExps(ymlMap map[string]interface{}) ([]string, bool) {
	result := getMapValueByPath("excludeRegExps", ymlMap)
	if result == nil {
		return nil, true
	} else {
		resultArray, ok := result.([]interface{})
		if !ok {
			fmt.Println("參數:excludeRegExps 非陣列")
			return nil, false
		}
		excludeRegExps := []string{}
		if len(resultArray) != 0 {
			for _, element := range resultArray {
				elementString, ok := element.(string)
				if ok {
					excludeRegExps = append(excludeRegExps, elementString)
				} else {
					fmt.Printf("陣列元素: %v 不是字串,請提供字串元素", element)
					return nil, false
				}
			}
		}
		fmt.Println("參數:excludeRegExps:", excludeRegExps)
		return excludeRegExps, true
	}
}

func deleteExcludePaths(somePath string, excludeRegExps []string) bool {
	if len(excludeRegExps) != 0 {

		error := filepath.Walk(somePath, func(path string, info fs.FileInfo, err error) error {
			if err == nil {
				positiveProjPath := filepath.ToSlash(path)
				for _, regExp := range excludeRegExps {

					isMatch, error1 := regexp.MatchString(regExp, positiveProjPath)
					if error1 != nil {
						fmt.Printf("正規比對發生異常,正規:%s", regExp)
						return error1
					}

					if isMatch {
						fmt.Printf("正規:%s,路徑:%s", regExp, path)
						fmt.Println()
						error2 := os.RemoveAll(path)
						if error2 != nil {
							return error2
						}
						if info.IsDir() {
							return filepath.SkipDir
						}
						break
					}
				}
			}
			return err
		})

		if error != nil {
			fmt.Printf("排除檔案發生錯誤: %v", error)
			return false
		}

	}
	return true
}

func getWorkspace() (string, bool) {
	var workspaceDir = os.Getenv(WORKSPACE)
	if workspaceDir == "" {
		fmt.Println("未提供環境變數: WORKSPACE")
		return "", false
	}
	fmt.Println("WORKSPACE: ", workspaceDir)
	if !filepath.IsAbs(workspaceDir) {
		fmt.Println("請提供絕對路徑")
		return "", false
	}
	if !isDir(workspaceDir) {
		fmt.Println("此路徑非目錄,請提供目錄")
		return "", false
	}
	return workspaceDir, true
}

func getYmlMap(someDir string) (map[string]interface{}, bool) {
	ymlFilePath := ""
	filepath.Walk(someDir, func(path string, info fs.FileInfo, err error) error {
		if err == nil && !info.IsDir() && "fortify.yml" == info.Name() {
			ymlFilePath = path
			return errors.New("don't need run")
		}
		return err
	})
	if ymlFilePath == "" {
		fmt.Println("WORKSPACE中不存在fortify.yml檔案,請放置此檔在專案的任意位置")
		return nil, false
	}
	// 讀fortify.yml的內容抓出參數,可以在init的時候直接放到記憶體中
	ymlMap, readYmlError := readYmlToMap(ymlFilePath)
	if readYmlError != nil {
		fmt.Println("讀取fortify.yml發生錯誤", readYmlError)
		return nil, false
	}
	return ymlMap, true
}

func checkPathIsSafe(path string) bool {
	isMatch, _ := regexp.MatchString(`(^\.\./)|(/\.\./)|(/\.\.$)`, filepath.ToSlash(path))
	if isMatch {
		fmt.Println("此相對路徑不允許使用..")
	}
	return !isMatch
}

func getProjPath(workspace string, projRelativePath string) (string, bool) {
	projPath := filepath.Join(workspace, projRelativePath)
	if !isDir(projPath) {
		fmt.Printf("專案目錄: %s 不存在或非目錄", projPath)
		return "", false
	}
	fmt.Println("projPath:", projPath)
	return projPath, true
}

func main() {

	workspaceDir, ok := getWorkspace()
	if !ok {
		return
	}

	ymlMap, ok := getYmlMap(workspaceDir)
	if !ok {
		return
	}

	projRelativePath, ok := getProjRelativePath(ymlMap)
	if !ok {
		return
	}

	if !checkPathIsSafe(projRelativePath) {
		return
	}

	excludeRegExps, ok := getExcludeRegExps(ymlMap)
	if !ok {
		return
	}

	// proj目錄
	projPath, ok := getProjPath(workspaceDir, projRelativePath)
	if !ok {
		return
	}

	if !deleteExcludePaths(projPath, excludeRegExps) {
		return
	}

	// ssh 刪除舊的資料

	// scp 將專案放置到某地方

	// call url use jwt token

}
