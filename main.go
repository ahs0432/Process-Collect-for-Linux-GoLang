package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"math"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

var prevTotalTime int64 = 0

type ConcurrentMap struct {
	sync.RWMutex
	items map[string]int64
}

func (cm *ConcurrentMap) setItem(key string, val int64) {
	cm.Lock()
	cm.items[key] = val
	defer cm.Unlock()
}

var prevCPUTime ConcurrentMap = ConcurrentMap{items: map[string]int64{}}

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

func checkProcStat(pid string, totalTime int64) string {
	procStatFile, err := ioutil.ReadFile("/proc/" + pid + "/stat")

	if err != nil {
		fmt.Println(err.Error())
		return "0"
	}

	procStat := strings.Split(string(procStatFile), " ")
	if len(procStat) < 14 {
		return "0"
	}

	var cpuTime int64

	if tmpData, err := strconv.ParseInt(procStat[13], 10, 64); err != nil {
		fmt.Println(err.Error())
		return "0"
	} else {
		cpuTime = tmpData
	}

	if _, exists := prevCPUTime.items[pid]; !exists {
		prevCPUTime.setItem(pid, cpuTime)
		return "0"
	}

	tmpCPUTime := prevCPUTime.items[pid]
	prevCPUTime.setItem(pid, cpuTime)

	return strconv.Itoa(int(math.Floor(float64(cpuTime-tmpCPUTime) / float64(totalTime-prevTotalTime) * 100)))
}

// /proc/[pid]/cmdline 파일 전체 확인 진행
func checkProcStatus(unixMilli string, procName string, pid string, totalTime int64, wg *sync.WaitGroup) {
	procStatusFile, err := ioutil.ReadFile("/proc/" + pid + "/status")

	if err != nil {
		fmt.Println(err.Error())
		wg.Done()
		return
	}

	procStatus := string(procStatusFile)
	procStatusName := strings.Split(strings.Split(procStatus, "Name:\t")[1], "\n")[0]

	if procName != procStatusName {
		wg.Done()
		return
	}

	content := (unixMilli + "," + checkProcStat(pid, totalTime) + "," + changeBytes(strings.ReplaceAll(strings.Split(strings.Split(procStatus, "VmSize:\t")[1], "\n")[0], " ", "")) + ",")

	if commands := checkProcCommandFile(pid); commands == nil {
		wg.Done()
		return
	} else {
		content += (commands[0] + "," + commands[1] + ",")
	}

	content += (strings.Split(strings.Split(procStatus, "Pid:\t")[1], "\n")[0] + "," + strings.Split(strings.Split(procStatus, "PPid:\t")[1], "\n")[0] + "," + checkPasswdUser(strings.Split(strings.Split(strings.Split(procStatus, "Uid:\t")[1], "\n")[0], "\t")[1]))

	writeFile("testFile.csv", content)
	wg.Done()
}

func checkTotalTime() int64 {
	procStatFile, err := ioutil.ReadFile("/proc/stat")

	if err != nil {
		fmt.Println(err.Error())
		return 0
	}

	cpuInfo := strings.Split(strings.Split(strings.Split(string(procStatFile), "cpu  ")[1], "\n")[0], " ")
	totalTime := int64(0)

	for i := 0; i < len(cpuInfo)-1; i++ {
		tmp, err := strconv.ParseInt(cpuInfo[i], 10, 64)
		if err != nil {
			fmt.Println(err.Error())
			return 0
		}
		totalTime += tmp
	}

	return totalTime
}

// 파일 존재 여부 확인 후 파일 생성
func createFile(path string) {
	var _, err = os.Stat(path)

	if os.IsNotExist(err) {
		file, err := os.Create(path)
		if err != nil {
			fmt.Println(err.Error())
			return
		}

		file.WriteString("TIME,CPU,MEMORYBYTES,CMD1,CMD2,PID,PPID,USER\n")
		file.Sync()

		defer file.Close()
	}
}

// 파일에 내용 추가
func writeFile(path string, content string) {
	file, err := os.OpenFile(path, os.O_RDWR|os.O_APPEND, 0644)
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	io.WriteString(file, content+"\n")

	file.Sync()
	defer file.Close()
}

func checkProcData(procName string, unixMilli string) {
	createFile("testFile.csv")
	pidList := searchProcList()
	if len(pidList) == 0 {
		return
	}

	totalTime := checkTotalTime()
	if totalTime == 0 {
		return
	}

	wg := sync.WaitGroup{}
	wg.Add(1)

	go func() {
		defer wg.Done()
		for _, pid := range pidList {
			wg.Add(1)
			go checkProcStatus(unixMilli, procName, pid, totalTime, &wg)
		}
	}()

	wg.Wait()
	prevTotalTime = totalTime
}

func main() {
	// 20 Second
	repeatTime := 20

	// 무한 루프를 걸어둔 뒤 특정 시점에 도달될 경우 Data Check 함수를 호출함
	for {
		time.Sleep(1 * time.Millisecond)
		unixMilli := time.Now().UnixMilli()

		if unixMilli%(int64(repeatTime)*1000) == 0 {
			checkProcData("httpd", strconv.FormatInt(unixMilli, 10))
		}
	}
}
