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
	"strings"
	"untitled/main/util"
)

const (
	WORKSPACE     = "WORKSPACE"
	MAIN_YML_PATH = "MAIN_YML_PATH"
)

func isDir(path string) bool {
	fileInfo, error := os.Stat(path)
	return error == nil && fileInfo.IsDir()
}

func isFile(path string) bool {
	fileInfo, error := os.Stat(path)
	return error == nil && !fileInfo.IsDir()
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
	if len(array) == 0 {
		return array
	}
	var newArray []string
	for _, element := range array {
		if element != "" {
			newArray = append(newArray, element)
		}
	}
	return newArray
}

func getNextObj(lastObj interface{}, element string) interface{} {
	if lastObj != nil {
		reflectValue := reflect.ValueOf(lastObj)
		if reflectValue.Kind() == reflect.Interface || reflectValue.Kind() == reflect.Ptr {
			reflectValue = reflectValue.Elem()
		}
		switch reflectValue.Kind() {
		case reflect.Map:
			reflectType := reflectValue.Type()
			elementValue := reflect.ValueOf(element)
			if reflectType.Key().AssignableTo(elementValue.Type()) {
				mapValue := reflectValue.MapIndex(elementValue)
				if mapValue.IsValid() {
					return mapValue.Interface()
				}
			}
			break
		case reflect.Slice:
			elementIndex, error := strconv.Atoi(element)
			if error == nil && reflectValue.Len() > elementIndex {
				return reflectValue.Index(elementIndex).Interface()
			}
			break
		}
	}
	return nil
}

func getMapValueByPath(mapPath string, someMap interface{}) interface{} {
	if someMap != nil {
		compile := regexp.MustCompile(`[\.\[\]]`)
		split := arrayNoEmpty(compile.Split(mapPath, -1))
		if len(split) != 0 {
			lastObj := someMap
			for _, element := range split {
				if lastObj = getNextObj(lastObj, element); lastObj == nil {
					return lastObj
				}
			}
			return lastObj
		}
	}
	return nil
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
						fmt.Printf("路徑:%s 正規:%s 比對相符", regExp, path)
						fmt.Println()
						error2 := os.RemoveAll(path)
						if error2 != nil {
							return error2
						}
						fmt.Println("刪除成功")
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
	workspaceDir := os.Getenv(WORKSPACE)
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

func getProjYmlMap(someDir string, ymlName string) (map[string]interface{}, bool) {
	ymlFilePath := ""
	filepath.Walk(someDir, func(path string, info fs.FileInfo, err error) error {
		if err == nil && !info.IsDir() && ymlName == info.Name() {
			ymlFilePath = path
			return errors.New("don't need run")
		}
		return err
	})
	if ymlFilePath == "" {
		fmt.Println("WORKSPACE中不存在fortify.yml檔案,請放置此檔在專案的任意位置")
		return nil, false
	}
	return getYmlMapByPath(ymlFilePath)
}

func getMainYmlMap() (map[string]interface{}, bool) {
	mainYmlPath := os.Getenv(MAIN_YML_PATH)
	if mainYmlPath == "" {
		fmt.Println("必須提供環境變數: MAIN_YML_PATH")
		return nil, false
	}
	return getYmlMapByPath(mainYmlPath)

}

func getYmlMapByPath(ymlPath string) (map[string]interface{}, bool) {
	if !isFile(ymlPath) {
		return nil, false
	}
	ymlMap, readYmlError := readYmlToMap(ymlPath)
	if readYmlError != nil {
		fmt.Printf("讀取yml檔發生錯誤 path:%s error:%v", ymlPath, readYmlError)
		fmt.Println()
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
	fmt.Println("參數 projPath:", projPath)
	return projPath, true
}

func combineTwoMap(firstMap interface{}, secondMap interface{}) {
	if firstMap != nil && secondMap != nil {
		firstMapRefValue := reflect.ValueOf(firstMap)
		secondMapRefValue := reflect.ValueOf(secondMap)
		if firstMapRefValue.Kind() == reflect.Interface || firstMapRefValue.Kind() == reflect.Ptr {
			firstMapRefValue = firstMapRefValue.Elem()
		}
		if secondMapRefValue.Kind() == reflect.Interface || secondMapRefValue.Kind() == reflect.Ptr {
			secondMapRefValue = secondMapRefValue.Elem()
		}
		if firstMapRefValue.Kind() == reflect.Map && secondMapRefValue.Kind() == reflect.Map {
			if secondMapRefValue.Len() != 0 {
				for _, secondMapRefValueKey := range secondMapRefValue.MapKeys() {
					firstMapRefValueValue := firstMapRefValue.MapIndex(secondMapRefValueKey)
					secondMapRefValueValue := secondMapRefValue.MapIndex(secondMapRefValueKey)
					if firstMapRefValueValue.IsValid() { // 有這個key
						if !firstMapRefValueValue.IsNil() && !secondMapRefValueValue.IsNil() {
							if firstMapRefValueValue.Kind() == reflect.Interface || firstMapRefValueValue.Kind() == reflect.Ptr {
								firstMapRefValueValue = firstMapRefValueValue.Elem()
							}
							if secondMapRefValueValue.Kind() == reflect.Interface || secondMapRefValueValue.Kind() == reflect.Ptr {
								secondMapRefValueValue = secondMapRefValueValue.Elem()
							}
							if firstMapRefValueValue.Kind() == reflect.Map && secondMapRefValueValue.Kind() == reflect.Map {
								combineTwoMap(firstMapRefValueValue.Interface(), secondMapRefValueValue.Interface())
								continue
							}
						}
					}
					firstMapRefValue.SetMapIndex(secondMapRefValueKey, secondMapRefValueValue)
				}
			}
		}

	}
}

type SshDeleteOldProjDirRunner struct {
	util.CmdModel
	HostnameOrIp string
	UserName     string
	DeletePaths  []string
	MkdirPath    string
}

func (s *SshDeleteOldProjDirRunner) AfterCallCmdDoing() {

}

func (s *SshDeleteOldProjDirRunner) IsFinishCondition(exitCode int) bool {
	return exitCode == 0
}

func (s *SshDeleteOldProjDirRunner) SetEnvAndCommand(operationSystem util.OperationSystem) {
	var mkdirCmd string
	if s.MkdirPath != "" {
		mkdirCmd = " && mkdir " + s.MkdirPath + "\""
	} else {
		mkdirCmd = "\""
	}
	cmdCommand := "\"rm -rf " + strings.Join(s.DeletePaths, " ") + mkdirCmd
	s.AddCommand("ssh", s.UserName+"@"+s.HostnameOrIp, cmdCommand)
}

func sshAndDeleteOldData(hostName string, userName string, deletePaths []string, mkdirPath string) *SshDeleteOldProjDirRunner {
	sshDeleteOldProjDirRunner := new(SshDeleteOldProjDirRunner)
	sshDeleteOldProjDirRunner.HostnameOrIp = hostName
	sshDeleteOldProjDirRunner.UserName = userName
	sshDeleteOldProjDirRunner.DeletePaths = deletePaths
	sshDeleteOldProjDirRunner.MkdirPath = mkdirPath
	util.CallCmd(sshDeleteOldProjDirRunner)
	return sshDeleteOldProjDirRunner
}

type ScpCopyProjRunner struct {
	util.CmdModel
	ProjPath    string
	SshHostname string
	SshUsername string
	TargetPath  string
}

func (s *ScpCopyProjRunner) AfterCallCmdDoing() {
}

func (s *ScpCopyProjRunner) IsFinishCondition(exitCode int) bool {
	return exitCode == 0
}

func (s *ScpCopyProjRunner) SetEnvAndCommand(operationSystem util.OperationSystem) {
	s.AddCommand("scp", "-r", s.ProjPath, s.SshUsername+"@"+s.SshHostname+":"+s.TargetPath)
}

func scpAndCopyDataToShareFolder(hostName string, userName string, projPath string, targetPath string) *ScpCopyProjRunner {
	scpCopyProjRunner := new(ScpCopyProjRunner)
	scpCopyProjRunner.SshHostname = hostName
	scpCopyProjRunner.SshUsername = userName
	scpCopyProjRunner.ProjPath = projPath
	scpCopyProjRunner.TargetPath = targetPath
	util.CallCmd(scpCopyProjRunner)
	return scpCopyProjRunner
}

func main() {

	fmt.Println("開始")

	workspaceDir, ok := getWorkspace()
	if !ok {
		return
	}

	mainYmlMap, ok := getMainYmlMap()
	if !ok {
		return
	}

	fmt.Println("mainYmlMap:", mainYmlMap)

	projYmlMap, ok := getProjYmlMap(workspaceDir, "fortify.yml")
	if !ok {
		return
	}
	fmt.Println("projYmlMap:", projYmlMap)
	combineTwoMap(mainYmlMap, projYmlMap)
	fmt.Println("合併ymlMap:", mainYmlMap)

	ymlInfo, ok := getYmlInfoByMapAndCheckIsFieldsOk(mainYmlMap, workspaceDir)
	if !ok {
		return
	}

	if !deleteExcludePaths(ymlInfo.ProjPath, ymlInfo.ExcludeRegExps) {
		return
	}

	sourceCodeDir := filepath.Join("/source_codes", ymlInfo.SshUsername, ymlInfo.ProjName)
	reportDir := filepath.Join("/source_codes", ymlInfo.SshUsername, "Report", ymlInfo.ProjName)

	fmt.Printf("ssh 刪除遠端 sourceCodeDir:%s reportDir:%s", sourceCodeDir, reportDir)
	fmt.Println()

	sshDeleteOldProjDirRunner := sshAndDeleteOldData(ymlInfo.SshHost, ymlInfo.SshUsername, []string{sourceCodeDir, reportDir}, sourceCodeDir)
	if !sshDeleteOldProjDirRunner.IsSuccess {
		fmt.Println("刪除失敗")
		return
	}

	targetDir := filepath.Join("/source_codes", ymlInfo.SshUsername)
	// scp 將專案放置到某地方
	scpCopyProjRunner := scpAndCopyDataToShareFolder(ymlInfo.SshHost, ymlInfo.SshUsername, ymlInfo.ProjPath, targetDir)
	if !scpCopyProjRunner.IsSuccess {
		fmt.Println("複製成功")
	}

	fmt.Println("結束")
}

type YmlInfo struct {
	ProjRelativePath string   "proj.relativePath"
	ExcludeRegExps   []string "proj.excludeRegExps"
	SshHost          string   "ssh.host"
	SshUsername      string   "ssh.username"
	ProjName         string
	ProjPath         string
}

func getYmlInfoByMapAndCheckIsFieldsOk(ymlMap map[string]interface{}, workspaceDir string) (returnYmlInfo *YmlInfo, isSuccess bool) {
	defer func() {
		if recover() != nil {
			returnYmlInfo = nil
			isSuccess = false
		}
	}()

	ymlInfo := new(YmlInfo)
	infoRefValue := reflect.ValueOf(ymlInfo).Elem()
	infoRefType := infoRefValue.Type()
	for i := 0; i < infoRefValue.NumField(); i++ {
		fieldRefType := infoRefType.Field(i)
		// 抓ymlpath
		ymlPath := string(fieldRefType.Tag)
		if ymlPath != "" {
			result := getMapValueByPath(ymlPath, ymlMap)
			if result != nil {

				resultRefValue := reflect.ValueOf(result)
				fieldRefValue := infoRefValue.Field(i)

				fieldType := fieldRefType.Type
				resultRefType := resultRefValue.Type()

				if fieldType.AssignableTo(resultRefType) {
					fieldRefValue.Set(resultRefValue)
				} else if fieldRefValue.Kind() == reflect.Slice && resultRefValue.Kind() == reflect.Slice {
					for i := 0; i < resultRefValue.Len(); i++ {
						resultElementRefValue := resultRefValue.Index(i)
						if resultElementRefValue.IsValid() {
							if !resultElementRefValue.IsNil() && (resultElementRefValue.Kind() == reflect.Interface || resultElementRefValue.Kind() == reflect.Ptr) {
								resultElementRefValue = resultElementRefValue.Elem()
							}
							fieldRefValue.Set(reflect.Append(fieldRefValue, resultElementRefValue))
						} else {
							return nil, false
						}
					}
				}
			}
		}
	}

	if ymlInfo.ProjRelativePath == "" {
		fmt.Println("yml須提供參數 proj.relativePath")
		return nil, false
	}
	fmt.Println("參數 proj.relativePath:", ymlInfo.ProjRelativePath)
	fmt.Println("參數 proj.excludeRegExps:", ymlInfo.ExcludeRegExps)

	if !checkPathIsSafe(ymlInfo.ProjRelativePath) {
		return nil, false
	}

	// proj目錄
	projPath, ok := getProjPath(workspaceDir, ymlInfo.ProjRelativePath)
	if !ok {
		return nil, false
	}
	ymlInfo.ProjPath = projPath

	// projName
	ymlInfo.ProjName = filepath.Base(projPath)

	if ymlInfo.SshHost == "" {
		fmt.Println("參數 ssh.host 未提供")
		return nil, false
	}
	fmt.Println("參數 ssh.host: ", ymlInfo.SshHost)

	if ymlInfo.SshUsername == "" {
		fmt.Println("參數 ssh.username 未提供")
		return nil, false
	}
	fmt.Println("參數 ssh.username:", ymlInfo.SshUsername)

	return ymlInfo, true
}
