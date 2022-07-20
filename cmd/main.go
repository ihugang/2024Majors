package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"regexp"

	"github.com/tidwall/pretty"
	"github.com/xuri/excelize/v2"
)

type ProvinceInfo struct {
	Name string
	Url  string
}

type SchoolInfo struct {
	Name string
	Url  string
	Code string
	Area string
}

type MajorInfo struct {
	Area       string
	SchoolName string
	Name       string
	Subjects   string
	Level      string
	Requests   string
}

var provincesEntry = []ProvinceInfo{}
var schoolsEntry = []SchoolInfo{}
var majorsEntry = []MajorInfo{}

const patternProvincesBlock = `(?sU)<table class="linner"[^>]*>(.*)<\/table>`
const patternHref = `<a href='(area_\d+\.html)'\s*[^>]*>(.*)<\/a>`
const patternHref2 = `<a href="([^>]*)" target="_blank"\s*>(.*)<\/a>`

const patternSchoolsBlock = `(?sU)<table class="lsch"[^>]*>(.*)<\/table>`
const patternSchoolTr = `(?sU)<tr [^>]*>(.*)<\/tr>`
const patternSchoolTd = `(?sU)<td>(.*)<\/td>\s*<td>(.*)<\/td>\s*<td>(.*)<\/td>\s*<td>(.*)<\/td>`

const patternSchoolMajorsBlock = `(?sU)<table width="100%" class="lgoto"[^>]*>(.*)<\/table>`
const patternMajorTr = `(?sU)<tr [^>]*>(.*)<\/tr>`
const patternMajorTd = `(?sU)<td [^>]*>(.*)<\/td>\s*<td [^>]*>(.*)<\/td>\s*<td [^>]*>(.*)<\/td>\s*<td [^>]*>(.*)<\/td>`

func main() {
	println("【2024年普通高校招生专业选考科目要求】资料导出工具v0.1")

	_, err := os.Stat("majors.json")
	if err != nil {
		if os.IsNotExist(err) {
			println("majors.json not exists")
			getAllMajors()
		}
	}

	Json2Excel()
	os.Exit(0)
}

////////////////////////////////
// 读取所有专业并存储为json文件
////////////////////////////////
func getAllMajors() {
	initUrl := "https://zt.zjzs.net/xk2024/"
	res, err := http.Get(initUrl)

	if err != nil {
		fmt.Fprintf(os.Stderr, "fetch: %v\n", err)
		os.Exit(1)
	}

	// 读取资源数据 body: []byte
	body, err := ioutil.ReadAll(res.Body)

	// 关闭资源流
	res.Body.Close()

	if err != nil {
		fmt.Fprintf(os.Stderr, "fetch: reading %s: %v\n", initUrl, err)
		os.Exit(1)
	}

	// 打印资源数据
	initContent := string(body)
	//fmt.Printf("%s", initContent)

	// 正则匹配
	// 匹配省份
	re := regexp.MustCompile(patternProvincesBlock)

	result := re.FindString(initContent)
	//fmt.Printf("\n\rfind:%s", result)

	re = regexp.MustCompile(patternHref)

	provinces := re.FindAllStringSubmatch(result, -1)
	//fmt.Printf("\n\rfind:%v", provinces)
	for _, province := range provinces {
		fmt.Printf("\n\rprovince:%s %s", province[2], province[1])
		provincesEntry = append(provincesEntry, ProvinceInfo{Name: province[2], Url: "https://zt.zjzs.net/xk2024/" + province[1]})
	}

	fmt.Printf("\n\rprovinces:%v", len(provincesEntry))

	// 匹配学校
	for _, province := range provincesEntry {
		fmt.Printf("\n now process %s", province.Name)
		url := province.Url

		// 打印资源数据
		provinceContent, err := ReadUrl(url)
		if err != nil {
			fmt.Fprintf(os.Stderr, "fetch: reading %s: %v\n", url, err)
			os.Exit(1)
		}

		re := regexp.MustCompile(patternSchoolsBlock)

		result := re.FindString(provinceContent)

		re = regexp.MustCompile(patternSchoolTr)

		schools := re.FindAllString(result, -1)
		fmt.Printf("\n\rfind:%d", len(schools))

		index := 0
		for _, school := range schools {
			// if index == 0 {
			// 	index += 1
			// 	continue
			// } else {
			// 	index += 1
			// }

			re = regexp.MustCompile(patternSchoolTd)

			infos := re.FindStringSubmatch(school)

			re = regexp.MustCompile(patternHref2)
			fmt.Printf("%s", infos[4])
			href := re.FindStringSubmatch(infos[4])

			aSchool := SchoolInfo{
				Name: infos[3],
				Url:  "https://zt.zjzs.net/xk2024/" + href[1],
				Code: infos[2],
				Area: infos[1],
			}

			schoolsEntry = append(schoolsEntry, aSchool)
			fmt.Printf("\n\r%d %s", index, aSchool)

			getSchoolMajors(aSchool.Url, aSchool.Name, province.Name)
			//os.Exit(0)
		}

	}

	jsonFilename := "majors.json"
	jsonFile, err := os.Create(jsonFilename)
	if err != nil {
		log.Println("Create JSON file error:", err)
		os.Exit(1)
	}
	w := bufio.NewWriter(jsonFile)
	data, err := json.Marshal(majorsEntry)
	if err != nil {
		err = fmt.Errorf("json.marshal failed, err:%s", err)
		log.Println(err)
	}
	jsonContent := pretty.Pretty(data)
	bytes, err := w.WriteString(string(jsonContent))
	if err != nil {
		log.Println("write json file error:", err)
	}
	w.Flush()
	jsonFile.Close()
	fmt.Printf("\n\rwrite json file success, bytes:%d", bytes)
}

/////////////////////////////////
// 读取学校的专业信息
/////////////////////////////////
func getSchoolMajors(url string, schoolName string, provinceName string) {
	// 打印资源数据
	fmt.Printf("\n now process school %s", schoolName)
	content, err := ReadUrl(url)
	if err != nil {
		fmt.Fprintf(os.Stderr, "fetch: reading %s: %v\n", url, err)
		os.Exit(1)
	}

	re := regexp.MustCompile(patternSchoolMajorsBlock)

	result := re.FindString(content)
	//fmt.Printf("\n\rfind:%s", result)
	re = regexp.MustCompile(patternMajorTr)

	majors := re.FindAllString(result, -1)
	//fmt.Printf("\n\rfind:%d", len(majors))
	index := 0
	for _, major := range majors {
		if index == 0 {
			index += 1
			continue
		} else {
			index += 1
		}

		re = regexp.MustCompile(patternMajorTd)

		infos := re.FindStringSubmatch(major)

		aMajor := MajorInfo{
			Area:       provinceName,
			SchoolName: schoolName,
			Name:       infos[2],
			Subjects:   infos[4],
			Level:      infos[1],
			Requests:   infos[3],
		}

		majorsEntry = append(majorsEntry, aMajor)
		fmt.Printf("\n\r%d %s", index, aMajor)
	}
}

////////////////////////////////////////////////////////////////
// ReadUrl - 读取网址内容
////////////////////////////////////////////////////////////////
func ReadUrl(url string) (string, error) {
	res, err := http.Get(url)

	if err != nil {
		fmt.Fprintf(os.Stderr, "fetch: %v\n", err)
		return "", err
	}

	// 读取资源数据 body: []byte
	body, err := ioutil.ReadAll(res.Body)

	// 关闭资源流
	res.Body.Close()

	if err != nil {
		fmt.Fprintf(os.Stderr, "fetch: reading %s: %v\n", url, err)
		return "", err
	}

	return string(body), nil
}

func Json2Excel() {
	filename := "majors.json"
	content, err := os.ReadFile(filename)
	if err != nil {
		log.Println(err)
		return
	}

	var majors []MajorInfo
	err = json.Unmarshal(content, &majors)
	if err != nil {
		log.Println(err)
		return
	}

	f := excelize.NewFile()

	f.SetCellValue("Sheet1", "A1", "省份")
	f.SetCellValue("Sheet1", "B1", "院校名称")
	f.SetCellValue("Sheet1", "C1", "专业（类）名称")
	f.SetCellValue("Sheet1", "D1", "类中所含专业")
	f.SetCellValue("Sheet1", "E1", "层次")
	f.SetCellValue("Sheet1", "F1", "选科科目要求")

	lineNo := 2
	for _, major := range majors {
		fmt.Printf("\n\r%s", major)
		f.SetCellValue("Sheet1", fmt.Sprintf("A%d", lineNo), major.Area)
		f.SetCellValue("Sheet1", fmt.Sprintf("B%d", lineNo), major.SchoolName)
		f.SetCellValue("Sheet1", fmt.Sprintf("C%d", lineNo), major.Name)
		f.SetCellValue("Sheet1", fmt.Sprintf("D%d", lineNo), major.Subjects)
		f.SetCellValue("Sheet1", fmt.Sprintf("E%d", lineNo), major.Level)
		f.SetCellValue("Sheet1", fmt.Sprintf("F%d", lineNo), major.Requests)

		lineNo += 1
	}

	if err := f.SaveAs("2024年普通高校招生专业选考科目要求.xlsx"); err != nil {
		fmt.Println(err)
	}

	fmt.Println("done")
}
