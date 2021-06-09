package util

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"runtime"
	"strings"
)

type OperationSystem int

const (
	Linux OperationSystem = iota
	Windows
)

var (
	OPERRATION_SYSTEM          OperationSystem
	CALL_CMD_FINISH_CLOSE_FLAG string
	EXECUTION_FILE_NAME        string
	SET_ENV_SYXTAX             string
)

func init() {
	osString := runtime.GOOS
	if strings.Contains(osString, "window") {
		OPERRATION_SYSTEM = Windows
		EXECUTION_FILE_NAME = "cmd"
		CALL_CMD_FINISH_CLOSE_FLAG = "/C"
		SET_ENV_SYXTAX = "set "
	} else {
		OPERRATION_SYSTEM = Linux
		EXECUTION_FILE_NAME = "sh"
		CALL_CMD_FINISH_CLOSE_FLAG = "-c"
		SET_ENV_SYXTAX = "export "
	}
}

type CmdModelInterface interface {
	AfterCallCmdDoing()                               // 成功失敗都會call
	IsFinishCondition(exitCode int) bool              // 由實做者判斷
	SetEnvAndCommand(operationSystem OperationSystem) // 由實做者實做
	SetIsSuccess(isSuccess bool)
	GetExecutePath() string
	GetRetryTimes() int
	GetReturnCode() int
	SetReturnCode(returnCode int)
	PutEnvMap(key, value string)
	AddCommand(commandOrCommandSplit ...string)
	GetCommandList() [][]string
	GetEnvMap() map[string]string
	GetResErrorMessagePointer() *[]string
	GetResCommonMessagePointer() *[]string
}

type CmdModel struct {
	ExecutePath      string            // 執行的目錄
	RetryTimes       int               // 要自己塞
	ReturnCode       int               //由util放
	EnvMap           map[string]string // 自己塞
	CommandList      [][]string        //自己塞
	ResCommonMessage []string          // util塞,但是重設之前要先清除
	ResErrorMessage  []string          // util塞,但是重設之前要先清除
	IsSuccess        bool
}

func (cmdModel *CmdModel) SetIsSuccess(IsSuccess bool) {
	cmdModel.IsSuccess = IsSuccess
}

func (cmdModel *CmdModel) GetExecutePath() string {
	return cmdModel.ExecutePath
}

func (cmdModel *CmdModel) GetRetryTimes() int {
	return cmdModel.RetryTimes
}

func (cmdModel *CmdModel) GetReturnCode() int {
	return cmdModel.ReturnCode
}

func (cmdModel *CmdModel) SetReturnCode(ReturnCode int) {
	cmdModel.ReturnCode = ReturnCode
}

func (cmdModel *CmdModel) PutEnvMap(key, value string) {
	if key != "" {
		if cmdModel.EnvMap == nil {
			cmdModel.EnvMap = map[string]string{}
		}
		cmdModel.EnvMap[key] = value
	}
}

func (cmdModel *CmdModel) AddCommand(cmds ...string) {
	if cmds != nil && len(cmds) != 0 {
		if cmdModel.CommandList == nil {
			cmdModel.CommandList = [][]string{}
		}
		cmdModel.CommandList = append(cmdModel.CommandList, cmds)
	}
}

func (cmdModel *CmdModel) GetEnvMap() map[string]string {
	return cmdModel.EnvMap
}

func (cmdModel *CmdModel) GetCommandList() [][]string {
	return cmdModel.CommandList
}

func (cmdModel *CmdModel) GetResCommonMessagePointer() *[]string {
	return &cmdModel.ResCommonMessage
}

func (cmdModel *CmdModel) GetResErrorMessagePointer() *[]string {
	return &cmdModel.ResErrorMessage
}

func CallCmd(cmdModel CmdModelInterface) CmdModelInterface {
	defer func() {
		err := recover()
		if err != nil {
			fmt.Println("err:", err)
		}
	}()
	cmdModel.SetEnvAndCommand(OPERRATION_SYSTEM)
	cmdModel.SetIsSuccess(false)
	for i := 0; i <= cmdModel.GetRetryTimes(); i++ {
		fmt.Printf("[CallCmd] call command 第%v次", i+1)
		fmt.Println()
		fullCommand := produceCommand(cmdModel.GetEnvMap(), cmdModel.GetCommandList())
		fmt.Println("[CallCmd] command:", fullCommand)
		cmd := exec.Command(EXECUTION_FILE_NAME, CALL_CMD_FINISH_CLOSE_FLAG, fullCommand)
		executePath := cmdModel.GetExecutePath()
		if executePath != "" {
			cmd.Dir = executePath
		}
		stdoutPipe, stdoutErr := cmd.StdoutPipe()
		if stdoutErr != nil {
			panic(stdoutErr)
		}
		stderrPipe, stderrErr := cmd.StderrPipe()
		if stderrErr != nil {
			panic(stderrErr)
		}

		go collectCmdReturnMessage(stdoutPipe, cmdModel.GetResCommonMessagePointer())
		go collectCmdReturnMessage(stderrPipe, cmdModel.GetResErrorMessagePointer())
		returnCode := getCmdExitCode(cmd.Run())
		cmdModel.SetReturnCode(returnCode)
		if cmdModel.IsFinishCondition(returnCode) {
			cmdModel.SetIsSuccess(true)
			cmdModel.AfterCallCmdDoing()
			break
		}
		cmdModel.AfterCallCmdDoing()
	}
	return cmdModel
}

func collectCmdReturnMessage(closer io.ReadCloser, collection *[]string) {
	*collection = []string{}
	scanner := bufio.NewScanner(closer)
	for scanner.Scan() {
		lineText := scanner.Text()
		fmt.Println(lineText)
		*collection = append(*collection, lineText)
	}
}

func getCmdExitCode(cmdErr error) int {
	if cmdErr == nil {
		return 0
	} else if exitError, ok := cmdErr.(*exec.ExitError); ok {
		return exitError.ExitCode()
	}
	panic(errors.New("unknowExitCode"))
}

func produceCommand(envMap map[string]string, cmdList [][]string) string {
	command := ""
	if envMap != nil && len(envMap) != 0 {
		isFirst := false
		for key, value := range envMap {
			if !isFirst {
				isFirst = true
			} else {
				command += "&&"
			}
			command += SET_ENV_SYXTAX + key + "=" + value
		}
	}

	if cmdList != nil && len(cmdList) != 0 {
		if command != "" {
			command += "&&"
		}
		for index, element := range cmdList {
			if index != 0 {
				command += "&&"
			}
			command += strings.Join(element, " ")
		}
	}

	return command
}
