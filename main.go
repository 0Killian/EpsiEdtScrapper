package main

import (
	"database/sql"
	"fmt"
	"github.com/ericchiang/css"
	"github.com/fxtlabs/date"
	_ "github.com/go-sql-driver/mysql"
	"github.com/joho/godotenv"
	"golang.org/x/net/html"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
	"io"
	"math"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

type Course struct {
	Date     string
	Category string
	Start    string
	End      string
	Subject  string
	Teacher  string
	Room     string
	Remote   bool
	Bts      bool
}

func main() {
	err := godotenv.Load()
	if err != nil {
		panic(fmt.Sprintf("error loading .env file: %s\n", err))
	}

	// Scrap all school year
	var courses []Course
	for i := 0; i < 52; i++ {
		d := date.New(2023, time.Month(9), 1).Add(i * 7)
		fmt.Printf("scraping week %s\n", d.Format("01/02/2006"))
		courses = addCourse(courses, "killian", "bellouard", d.Format("01/02/2006"))
		courses = addCourse(courses, "ewan", "lenogue", d.Format("01/02/2006"))
		courses = addCourse(courses, "enzo", "gourmelon", d.Format("01/02/2006"))
	}

	// Remove duplicates
	var coursesNoDup []Course
	for _, course := range courses {
		if !contains(coursesNoDup, course) {
			coursesNoDup = append(coursesNoDup, course)
		}
	}

	// Connect to database
	db, err := sql.Open("mysql", os.Getenv("DATABASE_URL_GO"))
	if err != nil {
		panic(fmt.Sprintf("error connecting to database: %s\n", err))
	}

	// Remove all courses
	_, err = db.Exec("DELETE FROM course")
	if err != nil {
		panic(fmt.Sprintf("error removing all courses: %s\n", err))
	}

	// Insert all courses
	for _, course := range coursesNoDup {
		_, err = db.Exec("INSERT INTO course (date, category, start, end, subject, teacher, classroom, remote, bts) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)", course.Date, course.Category, course.Start, course.End, course.Subject, course.Teacher, course.Room, course.Remote, course.Bts)
		if err != nil {
			panic(fmt.Sprintf("error inserting course: %s\n", err))
		}
	}
}

func addCourse(courses []Course, firstname string, lastname string, date string) []Course {
	for i := 0; i < 5; i++ {
		c := scrapWeek(firstname, lastname, date)
		if c == nil {
			continue
		}
		for _, course := range c {
			courses = append(courses, course)
		}
		return courses
	}
	panic("error scraping week")
}

func contains(dup []Course, course Course) bool {
	for _, c := range dup {
		if c.Date == course.Date && c.Category == course.Category && c.Start == course.Start && c.End == course.End && c.Subject == course.Subject && c.Teacher == course.Teacher && c.Room == course.Room && c.Remote == course.Remote && c.Bts == course.Bts {
			return true
		}
	}
	return false
}

func scrapWeek(firstname string, lastname string, date string) []Course {
	var courses []Course = []Course{}
	requestUrl := fmt.Sprintf("https://edtmobiliteng.wigorservices.net/WebPsDyn.aspx?action=posEDTBEECOME&serverid=i&Tel=%s.%s&date=%s", firstname, lastname, date)
	res, err := http.Get(requestUrl)

	if err != nil {
		fmt.Printf("error making http request: %s\n", err)
		return nil
	}

	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			fmt.Printf("error closing response body: %s\n", err)
		}
	}(res.Body)

	htmlRoot, err := html.Parse(res.Body)

	page := renderNode(htmlRoot)

	if strings.Contains(page, "Erreur") {
		fmt.Printf("error scraping week (%s): %s\n", requestUrl, page)
		return nil
	}

	daySel, err := css.Parse("div.Jour")

	if err != nil {
		fmt.Printf("error parsing css: %s\n", err)
		return nil
	}

	dayNodes := daySel.Select(htmlRoot)

	coursesSel, err := css.Parse("div.Case")
	if err != nil {
		fmt.Printf("error parsing css: %s\n", err)
		return nil
	}

	coursesNodes := coursesSel.Select(htmlRoot)

	tcjourSel, err := css.Parse(".TCJour")
	if err != nil {
		fmt.Printf("error parsing css: %s\n", err)
		return nil
	}

	tcjourNodes := tcjourSel.Select(htmlRoot)

	for dayIndex, dayNode := range dayNodes {
		dayLeft := getLeft(dayNode)

		for _, courseNode := range coursesNodes {
			courseLeft := getLeft(courseNode)
			if math.Floor(courseLeft) != math.Floor(dayLeft) {
				continue
			}

			day := strings.Split(tcjourNodes[dayIndex].FirstChild.Data, " ")

			dayDate, err := strconv.ParseInt(day[1], 10, 8)
			if err != nil {
				fmt.Printf("error parsing int: %s\n", err)
				return nil
			}
			dayMonth, err := strconv.ParseInt(getMonth(day[2]), 10, 8)
			if err != nil {
				fmt.Printf("error parsing int: %s\n", err)
				return nil
			}
			year, err := strconv.ParseInt(strings.Split(date, "/")[2], 10, 16)
			if err != nil {
				fmt.Printf("error parsing int: %s\n", err)
				return nil
			}

			elemSel, err := css.Parse(".innerCase > .BackGroundCase > table > tbody > tr > td.TChdeb")
			if err != nil {
				fmt.Printf("error parsing css: %s\n", err)
				return nil
			}

			duration := elemSel.Select(courseNode)[0]

			start := duration.FirstChild.Data[:5]
			end := duration.FirstChild.Data[8:13]

			subjectSel, err := css.Parse(".innerCase > .BackGroundCase > table > tbody > tr > td.TCase")
			if err != nil {
				fmt.Printf("error parsing css: %s\n", err)
				return nil
			}

			professorSel, err := css.Parse(".innerCase > .BackGroundCase > table > tbody > tr > td.TCProf")
			if err != nil {
				fmt.Printf("error parsing css: %s\n", err)
				return nil
			}

			roomSel, err := css.Parse(".innerCase > .BackGroundCase > table > tbody > tr > td.TCSalle")

			subject := renderNode(subjectSel.Select(courseNode)[0].FirstChild.NextSibling.NextSibling)
			professor := renderNode(professorSel.Select(courseNode)[0])
			bts := strings.Contains(professor, "BTS")

			professorLines := strings.Split(professor, "<br/>")

			professor = strings.Split(professorLines[0], "</span>")[1]
			category := strings.Split(professorLines[1], "</td>")[0]
			room := strings.Replace(roomSel.Select(courseNode)[0].FirstChild.Data, "Salle:", "", 1)
			remote := strings.Contains(strings.ToLower(room), "distanciel")

			var cat string

			if (strings.Contains(category, "dev") && strings.Contains(category, "infra")) || strings.Contains(category, "SIO") {
				cat = "devinfra"
			} else if strings.Contains(category, "dév") {
				cat = "dev"
			} else if strings.Contains(category, "infra") {
				cat = "infra"
			} else if strings.Contains(category, "relation") || strings.Contains(category, "NDRC") {
				cat = "marketing"
			} else {
				cat = "common"
			}

			courses = append(courses, Course{
				Date:     fmt.Sprintf("%d-%s-%s", year, strconv.Itoa(int(dayMonth)), strconv.Itoa(int(dayDate))),
				Category: cat,
				Start:    start,
				End:      end,
				Subject:  html.UnescapeString(subject),
				Teacher:  cases.Title(language.French).String(professor),
				Room:     room,
				Remote:   remote,
				Bts:      bts,
			})
		}
	}

	return courses
}

func getStyles(node *html.Node) string {
	var css string
	for _, attr := range node.Attr {
		if attr.Key == "style" {
			css = attr.Val
			break
		}
	}
	return css
}

func getLeft(node *html.Node) float64 {
	styles := getStyles(node)
	attributes := strings.Split(styles, ";")
	var left float64
	var err error
	for _, attr := range attributes {
		if strings.Contains(attr, "left") {
			leftStr := strings.Split(attr, ":")[1]
			left, err = strconv.ParseFloat(leftStr[:len(leftStr)-1], 32)

			if err != nil {
				fmt.Printf("error parsing float: %s\n", err)
				return 0
			}

			break
		}
	}

	return left
}

func searchChildrenClass(node *html.Node, class string) []*html.Node {
	var nodes []*html.Node
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		if hasClass(child, class) {
			nodes = append(nodes, child)
		}
	}
	return nodes
}

func searchChildrenElem(node *html.Node, elem string) []*html.Node {
	var nodes []*html.Node
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		if child.Data == elem {
			nodes = append(nodes, child)
		}
	}
	return nodes
}

func searchChildrenElemClass(node *html.Node, elem string, class string) []*html.Node {
	var nodes []*html.Node
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		if child.Data == elem && hasClass(child, class) {
			nodes = append(nodes, child)
		}
	}
	return nodes
}

func getClass(node *html.Node) string {
	var class string
	for _, attr := range node.Attr {
		if attr.Key == "class" {
			class = attr.Val
			break
		}
	}
	return class
}

func hasClass(node *html.Node, class string) bool {
	nodeClass := getClass(node)
	return strings.Contains(nodeClass, class)
}

func getMonth(date string) string {
	switch strings.ToLower(date) {
	case "janvier":
		return "01"
	case "février":
		return "02"
	case "mars":
		return "03"
	case "avril":
		return "04"
	case "mai":
		return "05"
	case "juin":
		return "06"
	case "juillet":
		return "07"
	case "août":
		return "08"
	case "septembre":
		return "09"
	case "octobre":
		return "10"
	case "novembre":
		return "11"
	case "décembre":
		return "12"
	default:
		return "00"
	}
}

func renderNode(node *html.Node) string {
	strWriter := &strings.Builder{}
	if err := html.Render(strWriter, node); err != nil {
		fmt.Printf("error rendering html: %s\n", err)
		return ""
	}
	return strWriter.String()
}
