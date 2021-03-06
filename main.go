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
	WORKSPACE   = "WORKSPACE"
	MainYmlPath = "MAIN_YML_PATH"
)

func isDir(path string) bool {
	fileInfo, err := os.Stat(path)
	return err == nil && fileInfo.IsDir()
}

func isFile(path string) bool {
	fileInfo, err := os.Stat(path)
	return err == nil && !fileInfo.IsDir()
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
			elementIndex, err := strconv.Atoi(element)
			if err == nil && reflectValue.Len() > elementIndex {
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

		err := filepath.Walk(somePath, func(path string, info fs.FileInfo, err error) error {
			if err == nil {
				positiveProjPath := filepath.ToSlash(path)
				for _, regExp := range excludeRegExps {

					isMatch, error1 := regexp.MatchString(regExp, positiveProjPath)
					if error1 != nil {
						fmt.Printf("????????????????????????,??????:%s", regExp)
						return error1
					}

					if isMatch {
						fmt.Printf("??????:%s ??????:%s ????????????", regExp, path)
						fmt.Println()
						error2 := os.RemoveAll(path)
						if error2 != nil {
							return error2
						}
						fmt.Println("????????????")
						if info.IsDir() {
							return filepath.SkipDir
						}
						break
					}
				}
			}
			return err
		})

		if err != nil {
			fmt.Printf("????????????????????????: %v", err)
			return false
		}

	}
	return true
}

func getWorkspace() (string, bool) {
	workspaceDir := os.Getenv(WORKSPACE)
	if workspaceDir == "" {
		fmt.Println("?????????????????????: WORKSPACE")
		return "", false
	}
	fmt.Println("WORKSPACE: ", workspaceDir)
	if !filepath.IsAbs(workspaceDir) {
		fmt.Println("?????????????????????")
		return "", false
	}
	if !isDir(workspaceDir) {
		fmt.Println("??????????????????,???????????????")
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
		fmt.Println("WORKSPACE????????????fortify.yml??????,???????????????????????????????????????")
		return nil, false
	}
	return getYmlMapByPath(ymlFilePath)
}

func getMainYmlMap() (map[string]interface{}, bool) {
	mainYmlPath := os.Getenv(MainYmlPath)
	if mainYmlPath == "" {
		fmt.Println("????????????????????????: MAIN_YML_PATH")
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
		fmt.Printf("??????yml??????????????? path:%s error:%v", ymlPath, readYmlError)
		fmt.Println()
		return nil, false
	}
	return ymlMap, true

}

func checkPathIsSafe(path string) bool {
	isMatch, _ := regexp.MatchString(`(^\.\./)|(/\.\./)|(/\.\.$)`, filepath.ToSlash(path))
	if isMatch {
		fmt.Println("??????????????????????????????..")
	}
	return !isMatch
}

func getProjPath(workspace string, projRelativePath string) (string, bool) {
	projPath := filepath.Join(workspace, projRelativePath)
	if !isDir(projPath) {
		fmt.Printf("????????????: %s ?????????????????????", projPath)
		return "", false
	}
	fmt.Println("?????? projPath:", projPath)
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
					if firstMapRefValueValue.IsValid() { // ?????????key
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

func (s *SshDeleteOldProjDirRunner) SetEnvAndCommand(util.OperationSystem) {
	cmdCommand := "\"rm -rf " + strings.Join(s.DeletePaths, " ")
	if s.MkdirPath != "" {
		cmdCommand += " && mkdir " + s.MkdirPath
	}
	cmdCommand += "\""
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

func (s *ScpCopyProjRunner) SetEnvAndCommand(util.OperationSystem) {
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

	fmt.Println("??????")

	workspaceDir, ok := getWorkspace()
	if !ok {
		os.Exit(1)
	}

	mainYmlMap, ok := getMainYmlMap()
	if !ok {
		os.Exit(1)
	}

	fmt.Println("mainYmlMap:", mainYmlMap)

	projYmlMap, ok := getProjYmlMap(workspaceDir, "fortify.yml")
	if !ok {
		os.Exit(1)
	}
	fmt.Println("projYmlMap:", projYmlMap)
	combineTwoMap(mainYmlMap, projYmlMap)
	fmt.Println("??????ymlMap:", mainYmlMap)

	ymlInfo, ok := getYmlInfoByMapAndCheckIsFieldsOk(mainYmlMap, workspaceDir)
	if !ok {
		os.Exit(1)
	}

	if !deleteExcludePaths(ymlInfo.ProjPath, ymlInfo.ExcludeRegExps) {
		os.Exit(1)
	}

	sourceCodeDir := filepath.Join("/source_codes", ymlInfo.SshUsername, ymlInfo.ProjName)
	reportDir := filepath.Join("/source_codes", ymlInfo.SshUsername, "Report", ymlInfo.ProjName)

	fmt.Printf("ssh ???????????? sourceCodeDir:%s reportDir:%s", sourceCodeDir, reportDir)
	fmt.Println()

	sshDeleteOldProjDirRunner := sshAndDeleteOldData(ymlInfo.SshHost, ymlInfo.SshUsername, []string{sourceCodeDir, reportDir}, sourceCodeDir)
	if !sshDeleteOldProjDirRunner.IsSuccess {
		fmt.Println("????????????")
		os.Exit(1)
	}

	targetDir := filepath.Join("/source_codes", ymlInfo.SshUsername)
	// scp ???????????????????????????
	scpCopyProjRunner := scpAndCopyDataToShareFolder(ymlInfo.SshHost, ymlInfo.SshUsername, ymlInfo.ProjPath, targetDir)
	if !scpCopyProjRunner.IsSuccess {
		fmt.Println("????????????")
		os.Exit(1)
	}

	fmt.Println("??????")
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
		// ???ymlpath
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
		fmt.Println("yml??????????????? proj.relativePath")
		return nil, false
	}
	fmt.Println("?????? proj.relativePath:", ymlInfo.ProjRelativePath)
	fmt.Println("?????? proj.excludeRegExps:", ymlInfo.ExcludeRegExps)

	if !checkPathIsSafe(ymlInfo.ProjRelativePath) {
		return nil, false
	}

	// proj??????
	projPath, ok := getProjPath(workspaceDir, ymlInfo.ProjRelativePath)
	if !ok {
		return nil, false
	}
	ymlInfo.ProjPath = projPath

	// projName
	ymlInfo.ProjName = filepath.Base(projPath)

	if ymlInfo.SshHost == "" {
		fmt.Println("?????? ssh.host ?????????")
		return nil, false
	}
	fmt.Println("?????? ssh.host: ", ymlInfo.SshHost)

	if ymlInfo.SshUsername == "" {
		fmt.Println("?????? ssh.username ?????????")
		return nil, false
	}
	fmt.Println("?????? ssh.username:", ymlInfo.SshUsername)

	return ymlInfo, true
}
