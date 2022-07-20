package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"sort"
	"strings"

	"github.com/tidwall/pretty"
	"github.com/xuri/excelize/v2"
)

type ProvinceInfo struct {
	Name string
	Url  string
}

type SchoolInfo struct {
	Name   string
	Url    string
	Code   string
	Area   string
	Majors []string
}

type SchoolMajorInfo struct {
	SchoolName string
	Majors     []string
}

type MajorInfo struct {
	Area          string
	SchoolName    string
	Name          string
	Subjects      string
	Level         string
	Requests      string
	Is211         bool
	Is985         bool
	IsLevelA      bool
	IsLevelB      bool
	IsLevelAMajor bool
}

var provincesEntry = []ProvinceInfo{}
var schoolsEntry = []SchoolInfo{}
var majorsEntry = []MajorInfo{}
var schoolMajorsEntry = []SchoolMajorInfo{}

const patternProvincesBlock = `(?sU)<table class="linner"[^>]*>(.*)<\/table>`
const patternHref = `<a href='(area_\d+\.html)'\s*[^>]*>(.*)<\/a>`
const patternHref2 = `<a href="([^>]*)" target="_blank"\s*>(.*)<\/a>`

const patternSchoolsBlock = `(?sU)<table class="lsch"[^>]*>(.*)<\/table>`
const patternSchoolTr = `(?sU)<tr [^>]*>(.*)<\/tr>`
const patternSchoolTd = `(?sU)<td>(.*)<\/td>\s*<td>(.*)<\/td>\s*<td>(.*)<\/td>\s*<td>(.*)<\/td>`

const patternSchoolMajorsBlock = `(?sU)<table width="100%" class="lgoto"[^>]*>(.*)<\/table>`
const patternMajorTr = `(?sU)<tr [^>]*>(.*)<\/tr>`
const patternMajorTd = `(?sU)<td [^>]*>(.*)<\/td>\s*<td [^>]*>(.*)<\/td>\s*<td [^>]*>(.*)<\/td>\s*<td [^>]*>(.*)<\/td>`

var _211Schools = []string{}
var _985Schools = []string{}
var _levelASchools = []string{}
var _levelBSchools = []string{}
var _levelAMajorSchools = []string{}

func main() {
	println("【2024年普通高校招生专业选考科目要求】资料导出工具v0.1")

	_211Schools, _ = getLevelOneSchooList()
	_985Schools, _ = get985SchoolList()
	_levelASchools, _ = get211SchoolList()
	_levelBSchools, _ = getLevelOneTypeBSchooList()
	_levelAMajorSchools, _ = getLevelOneMajorSchoolList()

	_, err := os.Stat("majors.json")
	if err != nil {
		if os.IsNotExist(err) {
			println("majors.json not exists")
			getAllMajors()
		}
	}

	getLevelOneMajorList()

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
		provinceContent, err := ReadUrl(url, false)
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
	content, err := ReadUrl(url, false)
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
func ReadUrl(urlPath string, useSSL bool) (string, error) {
	var client http.Client
	if useSSL {
		uri, err := url.Parse("http://127.0.0.1:4780")

		if err != nil {
			log.Fatal("parse url error: ", err)
		}

		client = http.Client{
			Transport: &http.Transport{
				// 设置代理
				Proxy: http.ProxyURL(uri),
			},
		}
	}

	res, err := client.Get(urlPath)

	if err != nil {
		fmt.Fprintf(os.Stderr, "fetch: %v\n", err)
		return "", err
	}

	// 读取资源数据 body: []byte
	body, err := ioutil.ReadAll(res.Body)

	// 关闭资源流
	res.Body.Close()

	if err != nil {
		fmt.Fprintf(os.Stderr, "fetch: reading %s: %v\n", urlPath, err)
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
	f.SetCellValue("Sheet1", "C1", "院校类型")
	f.SetCellValue("Sheet1", "D1", "专业特点")
	f.SetCellValue("Sheet1", "E1", "专业（类）名称")
	f.SetCellValue("Sheet1", "F1", "类中所含专业")
	f.SetCellValue("Sheet1", "G1", "层次")
	f.SetCellValue("Sheet1", "H1", "选科科目要求")

	lineNo := 2
	for _, major := range majors {
		fmt.Printf("\n\r%s", major.SchoolName+" "+major.Name)
		f.SetCellValue("Sheet1", fmt.Sprintf("A%d", lineNo), major.Area)
		f.SetCellValue("Sheet1", fmt.Sprintf("B%d", lineNo), major.SchoolName)
		f.SetCellValue("Sheet1", fmt.Sprintf("C%d", lineNo), getSchoolExtraInfo(major.SchoolName))
		f.SetCellValue("Sheet1", fmt.Sprintf("D%d", lineNo), getMajorExtraInfo(major.SchoolName, major.Name, major.Subjects))
		f.SetCellValue("Sheet1", fmt.Sprintf("E%d", lineNo), major.Name)
		f.SetCellValue("Sheet1", fmt.Sprintf("F%d", lineNo), major.Subjects)
		f.SetCellValue("Sheet1", fmt.Sprintf("G%d", lineNo), major.Level)
		f.SetCellValue("Sheet1", fmt.Sprintf("H%d", lineNo), major.Requests)

		lineNo += 1
	}

	if err := f.SaveAs("2024年普通高校招生专业选考科目要求.xlsx"); err != nil {
		fmt.Println(err)
	}

	fmt.Println("done")
}

func getMajorExtraInfo(schoolName string, major string, subs string) string {
	for _, school := range schoolMajorsEntry {
		if school.SchoolName == schoolName {
			for _, item := range school.Majors {
				if item == major || strings.Contains(item, major) || strings.Contains(major, item) || strings.Contains(subs, item) {
					return "一流学科-" + item
				}
			}

			break
		}
	}
	return ""
}

func getSchoolExtraInfo(name string) string {
	r := ""
	if isElementExist(_985Schools, name) {
		r = "985"
	}
	if isElementExist(_211Schools, name) {
		if len(r) == 0 {
			r = "211"
		} else {
			r = r + "、" + "211"
		}
	}

	r1 := ""
	if isElementExist(_levelASchools, name) {
		if len(r1) == 0 {
			r1 = "一流大学"
		} else {
			r1 = r1 + "、" + "一流大学"
		}
	}

	if isElementExist(_levelBSchools, name) {
		if len(r1) == 0 {
			r1 = "一流大学B类"
		} else {
			r1 = r1 + "、" + "一流大学B类"
		}
	}

	r2 := ""
	if isElementExist(_levelAMajorSchools, name) {
		r2 = "一流学科建设高校"
	}

	if len(r1) > 0 && len(r2) > 0 {
		if len(r) == 0 {
			r = "一流大学"
		} else {
			r = r + "、" + "一流大学"
		}
	} else if len(r1) > 0 {
		if len(r) == 0 {
			r = r1
		} else {
			r = r + "、" + r1
		}
	} else if len(r2) > 0 {
		if len(r) == 0 {
			r = r2
		} else {
			r = r + "、" + r2
		}
	}

	return r
}

func isElementExist(s []string, str string) bool {
	for _, v := range s {
		if v == str {
			return true
		}
	}
	return false
}

////////////////////////////////
// 获取211学校清单
////////////////////////////////
func get211SchoolList() ([]string, error) {
	fmt.Println("211 list:")
	const patternSchoolsBlock = `(?sU)<table class="MsoNormalTable"[^>]*>(.*)<\/table>`
	const patternTr = `(?sU)<tr [^>]*>(.*)<\/tr>`
	const patternTd = `(?sU)<td [^>]*>\s*<p [^>]*><span [^>]*>([^<>]*)<\/span>\s*<\/p>\s*<\/td>`

	url := "http://www.moe.gov.cn/srcsite/A22/s7065/200512/t20051223_82762.html"
	content, err := ReadUrl(url, false)
	if err != nil {
		fmt.Fprintf(os.Stderr, "fetch: reading %s: %v\n", url, err)
		os.Exit(1)
	}

	re := regexp.MustCompile(patternSchoolsBlock)
	result := re.FindString(content)
	//fmt.Println(result)
	re = regexp.MustCompile(patternTd)
	schools := re.FindAllStringSubmatch(result, -1)

	r := []string{}

	for _, school := range schools {
		fmt.Printf("%s\n\r", school[1])
		s := strings.TrimSpace(school[1])
		if len(s) > 0 {
			r = append(r, school[1])
		}
	}

	fmt.Println("")

	jsonFilename := "211.json"
	jsonFile, err := os.Create(jsonFilename)
	if err != nil {
		log.Println("Create JSON file error:", err)
		os.Exit(1)
	}
	w := bufio.NewWriter(jsonFile)
	data, err := json.Marshal(r)
	if err != nil {
		err = fmt.Errorf("json.marshal failed, err:%s", err)
		log.Println(err)
	}
	jsonContent := pretty.Pretty(data)
	_, err = w.WriteString(string(jsonContent))
	if err != nil {
		log.Println("write json file error:", err)
	}
	w.Flush()
	jsonFile.Close()

	return r, nil
}

////////////////////////////////
// 获取985学校清单
////////////////////////////////
func get985SchoolList() ([]string, error) {
	fmt.Println("985 list:")
	const patternSchoolsBlock = `(?sU)<table class="MsoNormalTable"[^>]*>(.*)<\/table>`
	const patternTd = `(?sU)<td [^>]*>\s*<p [^>]*><span [^>]*>([^<>]*)<\/span>\s*<\/p>\s*<\/td>`

	url := "http://www.moe.gov.cn/srcsite/A22/s7065/200612/t20061206_128833.html"
	content, err := ReadUrl(url, false)
	if err != nil {
		fmt.Fprintf(os.Stderr, "fetch: reading %s: %v\n", url, err)
		os.Exit(1)
	}

	re := regexp.MustCompile(patternSchoolsBlock)
	result := re.FindString(content)
	//fmt.Println(result)
	re = regexp.MustCompile(patternTd)
	schools := re.FindAllStringSubmatch(result, -1)

	r := []string{}

	for _, school := range schools {
		fmt.Printf("%s\n\r", school[1])
		s := strings.TrimSpace(school[1])
		if len(s) > 0 {
			r = append(r, school[1])
		}
	}

	if !isElementExist(r, "北京大学") {
		r = append(r, "北京大学")
	}

	jsonFilename := "985.json"
	jsonFile, err := os.Create(jsonFilename)
	if err != nil {
		log.Println("Create JSON file error:", err)
		os.Exit(1)
	}
	w := bufio.NewWriter(jsonFile)
	data, err := json.Marshal(r)
	if err != nil {
		err = fmt.Errorf("json.marshal failed, err:%s", err)
		log.Println(err)
	}
	jsonContent := pretty.Pretty(data)
	_, err = w.WriteString(string(jsonContent))
	if err != nil {
		log.Println("write json file error:", err)
	}
	w.Flush()
	jsonFile.Close()

	fmt.Println("")
	return r, nil
}

func getLevelOneSchooList() ([]string, error) {
	fmt.Println("一流大学A类名单:")
	content := "北京大学、中国人民大学、清华大学、北京航空航天大学、北京理工大学、中国农业大学、北京师范大学、中央民族大学、南开大学、天津大学、大连理工大学、吉林大学、哈尔滨工业大学、复旦大学、同济大学、上海交通大学、华东师范大学、南京大学、东南大学、浙江大学、中国科学技术大学、厦门大学、山东大学、中国海洋大学、武汉大学、华中科技大学、中南大学、中山大学、华南理工大学、四川大学、重庆大学、电子科技大学、西安交通大学、西北工业大学、兰州大学、国防科技大学"
	list := strings.Split(content, "、")
	r := []string{}
	for _, school := range list {
		fmt.Printf("%s\n\r", school)
		r = append(r, school)
	}
	fmt.Println("")
	return r, nil
}

func getLevelOneTypeBSchooList() ([]string, error) {
	fmt.Println("一流大学B类名单:")
	content := "东北大学、郑州大学、湖南大学、云南大学、西北农林科技大学、新疆大学"
	list := strings.Split(content, "、")
	r := []string{}
	for _, school := range list {
		fmt.Printf("%s\n\r", school)
		r = append(r, school)
	}
	fmt.Println("")
	return r, nil
}

func getLevelOneMajorSchoolList() ([]string, error) {
	fmt.Println("一流学科建设高校:")
	content := "北京交通大学、北京工业大学、北京科技大学、北京化工大学、北京邮电大学、北京林业大学、北京协和医学院、北京中医药大学、首都师范大学、北京外国语大学、中国传媒大学、中央财经大学、对外经济贸易大学、外交学院、中国人民公安大学、北京体育大学、中央音乐学院、中国音乐学院、中央美术学院、中央戏剧学院、中国政法大学、天津工业大学、天津医科大学、天津中医药大学、华北电力大学、河北工业大学、太原理工大学、内蒙古大学、辽宁大学、大连海事大学、延边大学、东北师范大学、哈尔滨工程大学、东北农业大学、东北林业大学、华东理工大学、东华大学、上海海洋大学、上海中医药大学、上海外国语大学、上海财经大学、上海体育学院、上海音乐学院、上海大学、苏州大学、南京航空航天大学、南京理工大学、中国矿业大学、南京邮电大学、河海大学、江南大学、南京林业大学、南京信息工程大学、南京农业大学、南京中医药大学、中国药科大学、南京师范大学、中国美术学院、安徽大学、合肥工业大学、福州大学、南昌大学、河南大学、中国地质大学、武汉理工大学、华中农业大学、华中师范大学、中南财经政法大学、湖南师范大学、暨南大学、广州中医药大学、华南师范大学、海南大学、广西大学、西南交通大学、西南石油大学、成都理工大学、四川农业大学、成都中医药大学、西南大学、西南财经大学、贵州大学、西藏大学、西北大学、西安电子科技大学、长安大学、陕西师范大学、青海大学、宁夏大学、石河子大学、中国石油大学、宁波大学、中国科学院大学、第二军医大学、第四军医大学"
	list := strings.Split(content, "、")
	r := []string{}
	for _, school := range list {
		fmt.Printf("%s\n\r", school)
		r = append(r, school)
	}
	fmt.Println("")
	return r, nil
}

func removeHtmlTag(in string) string {
	// regex to match html tag
	const pattern = `(<\/?[a-zA-A]+?[^>]*\/?>)*`
	r := regexp.MustCompile(pattern)
	groups := r.FindAllString(in, -1)
	// should replace long string first
	sort.Slice(groups, func(i, j int) bool {
		return len(groups[i]) > len(groups[j])
	})
	for _, group := range groups {
		if strings.TrimSpace(group) != "" {
			in = strings.ReplaceAll(in, group, "")
		}
	}
	return in
}

////////////////////////////////
/// 一流学科列表
////////////////////////////////
func getLevelOneMajorList() {
	fmt.Println("一流学科 list:")
	const patternMajorsBlock = `(?sU)<table class="wikitable[^>]*>(.*)“双一流”建设学科名单（按学校代码排序）(.*)<\/tbody>`
	const patternMajorsBlock2 = `(?sU)<tbody>(.*)<\/tbody>`
	const patternMajorTr = `(?sU)<tr>(.*)<\/tr>`
	const patternTd = `(?sU)<td>(.*)<\/td>`
	const patternMark = `<[^>]*>(.*)<\/[^>]*>`

	url := "https://zh.wikipedia.org/zh-hans/%E7%AC%AC%E4%B8%80%E8%BD%AE%E4%B8%96%E7%95%8C%E4%B8%80%E6%B5%81%E5%A4%A7%E5%AD%A6%E5%92%8C%E4%B8%80%E6%B5%81%E5%AD%A6%E7%A7%91%E5%BB%BA%E8%AE%BE%E9%AB%98%E6%A0%A1%E5%8F%8A%E5%BB%BA%E8%AE%BE%E5%AD%A6%E7%A7%91%E5%90%8D%E5%8D%95"
	content, err := ReadUrl(url, true)
	if err != nil {
		fmt.Fprintf(os.Stderr, "fetch: reading %s: %v\n", url, err)
		os.Exit(1)
	}

	re := regexp.MustCompile(patternMajorsBlock)
	result := re.FindString(content)

	re = regexp.MustCompile(patternMajorsBlock2)
	result = re.FindString(result)

	//fmt.Println("\n\r2:", result)
	re = regexp.MustCompile(patternMajorTr)
	schools := re.FindAllStringSubmatch(result, -1)

	index := 0
	for _, school := range schools {
		if index < 2 {
			index += 1
			continue
		}
		fmt.Printf("%s\n\r", school[1])
		re = regexp.MustCompile(patternTd)
		infos := re.FindAllStringSubmatch(school[1], -1)
		fmt.Println("\n\r3:", infos)
		schoolName := strings.Replace(removeHtmlTag(infos[0][1]), "\n", "", -1)
		majorInfo := strings.TrimSpace(removeHtmlTag(infos[2][1]))
		majors := strings.Split(majorInfo, "、")
		majors2 := []string{}
		for _, major := range majors {
			majors2 = append(majors2, strings.Replace(major, "（自定）", "", -1))
		}

		sm := SchoolMajorInfo{
			SchoolName: schoolName,
			Majors:     majors2,
		}

		schoolMajorsEntry = append(schoolMajorsEntry, sm)

		//fmt.Printf("%s \n\r %s \n\r", schoolName, majors)
	}

	jsonFilename := "levelmajors.json"
	jsonFile, err := os.Create(jsonFilename)
	if err != nil {
		log.Println("Create JSON file error:", err)
		return
	}
	w := bufio.NewWriter(jsonFile)
	data, err := json.Marshal(schoolMajorsEntry)
	if err != nil {
		err = fmt.Errorf("json.marshal failed, err:%s", err)
		log.Println(err)
		return
	}
	jsonContent := pretty.Pretty(data)
	_, err = w.WriteString(string(jsonContent))
	if err != nil {
		log.Println("write json file error:", err)
		return
	}
	w.Flush()
	jsonFile.Close()

	fmt.Println("")
}
