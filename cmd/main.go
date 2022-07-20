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
const patternHref = `<a href='([^>]*)'\s*>(.*)<\/a>`
const patternHref2 = `<a href="([^>]*)" target="_blank"\s*>(.*)<\/a>`

const patternSchoolsBlock = `(?sU)<table class="lsch"[^>]*>(.*)<\/table>`
const patternSchoolTr = `(?sU)<tr [^>]*>(.*)<\/tr>`
const patternSchoolTd = `(?sU)<td>(.*)<\/td>\s*<td>(.*)<\/td>\s*<td>(.*)<\/td>\s*<td>(.*)<\/td>`

const patternSchoolMajorsBlock = `(?sU)<table width="100%" class="lgoto"[^>]*>(.*)<\/table>`
const patternMajorTr = `(?U)<tr [^>]*>(.*)<\/tr>`
const patternMajorTd = `(?sU)<td [^>]*>(.*)<\/td>\s*<td [^>]*>(.*)<\/td>\s*<td [^>]*>(.*)<\/td>\s*<td [^>]*>(.*)<\/td>`

func main() {
	println("Hello, world!")

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
			if index == 0 {
				index += 1
				continue
			} else {
				index += 1
			}

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
			os.Exit(0)
			getSchoolMajors(aSchool.Url, province.Name, aSchool.Name)
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
	content, err := ReadUrl(url)
	if err != nil {
		fmt.Fprintf(os.Stderr, "fetch: reading %s: %v\n", url, err)
		os.Exit(1)
	}

	re := regexp.MustCompile(patternSchoolMajorsBlock)

	result := re.FindString(content)

	re = regexp.MustCompile(patternMajorTr)

	majors := re.FindAllString(result, -1)

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
