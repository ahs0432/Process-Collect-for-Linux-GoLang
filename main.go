package main

import (
	"fmt"
	"io/ioutil"
	"strconv"
	"strings"
	"time"
)

// Integer Check
func isInteger(s string) bool {
	if _, err := strconv.Atoi(s); err != nil {
		return false
	}

	return true
}

// /proc 경로 내 파일 검색 및 []string 형태로 전달
func searchProcList() []string {
	files, err := ioutil.ReadDir("/proc")

	if err != nil {
		fmt.Println(err.Error())
		return []string{}
	}

	pidList := []string{}

	for _, file := range files {
		if isInteger(file.Name()) {
			pidList = append(pidList, file.Name())
		}
	}

	return pidList
}

// /proc/[pid]/cmdline 파일 확인하여 커맨드 전달
func checkProcCommandFile(pid string) []string {
	procCommandFile, err := ioutil.ReadFile("/proc/" + pid + "/cmdline")

	if err != nil {
		fmt.Println(err.Error())
		return []string{}
	}

	procCommand := strings.Split(string(procCommandFile), "\000")
	return []string{procCommand[0], strings.Join(procCommand[1:], " ")}
}

// /etc/passwd 파일 확인하여 UID 비교 후 데이터 전달
func checkPasswdUser(uid string) string {
	passwdUserFile, err := ioutil.ReadFile("/etc/passwd")

	if err != nil {
		fmt.Println(err.Error())
		return ""
	}

	s := strings.Split(strings.Split(string(passwdUserFile), ":x:"+uid+":")[0], "\n")
	return s[len(s)-1]
}

// VmSize로 가져온 현재 점유 Memory를 Byte 단위로 변경
func changeBytes(s string) string {
	checkString := ""
	multi := 1

	if len(strings.Split(s, "kB")) == 2 {
		checkString = "kB"
		multi = 1024
	} else if len(strings.Split(s, "mB")) == 2 {
		checkString = "mB"
		multi = 1024 * 1024
	} else if len(strings.Split(s, "gB")) == 2 {
		checkString = "gB"
		multi = 1024 * 1024 * 1024
	} else if len(strings.Split(s, "tB")) == 2 {
		checkString = "gB"
		multi = 1024 * 1024 * 1024 * 1024
	}

	tmp, err := strconv.Atoi(strings.Split(s, checkString)[0])

	if err != nil {
		fmt.Println(err.Error())
		return "0"
	}

	return strconv.Itoa(tmp * multi)
}

// /proc/[pid]/cmdline 파일 전체 확인 진행
func checkProcStatus(unixMilli string, procName string) []string {
	pidList := searchProcList()
	if len(pidList) == 0 {
		return nil
	}

	contents := []string{}

	for _, pid := range pidList {
		procStatusFile, err := ioutil.ReadFile("/proc/" + pid + "/status")

		if err != nil {
			fmt.Println(err.Error())
			continue
		}

		procStatus := string(procStatusFile)
		procStatusName := strings.Split(strings.Split(procStatus, "Name:\t")[1], "\n")[0]

		if procName != procStatusName {
			continue
		}

		content := (unixMilli + "," + changeBytes(strings.ReplaceAll(strings.Split(strings.Split(procStatus, "VmSize:\t")[1], "\n")[0], " ", "")) + ",")

		if commands := checkProcCommandFile(pid); commands == nil {
			continue
		} else {
			content += (commands[0] + "," + commands[1])
		}

		content += (strings.Split(strings.Split(procStatus, "Pid:\t")[1], "\n")[0] + "," + strings.Split(strings.Split(procStatus, "PPid:\t")[1], "\n")[0] + "," + checkPasswdUser(strings.Split(strings.Split(strings.Split(procStatus, "Uid:\t")[1], "\n")[0], "\t")[1]))

		contents = append(contents, content)
	}

	return contents
}

func checkProcData(procName string) {
	contents := checkProcStatus(strconv.FormatInt(time.Now().UnixMilli(), 10), procName)

	fmt.Println(contents)
}

func main() {
	checkProcData("httpd")
}
