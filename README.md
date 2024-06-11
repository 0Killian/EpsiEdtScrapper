# EpsiEdtScrapper
A simple tool used to retrieve the schedule of a class from the EPSI schedule.

## Prerequisites
- Go
- A SQL database (tested with MySQL). The tables needed are created by the main backend of this project, [PSN2-EDT](https://github.com/0Killian/PSN2-EDT), and are available in the `migrations` folder.

## Usage
You must first specify the users you want to retrieve the schedule from. Currently, it is hardcoded in the `main.go` file (line 45). For each user, you must add a call to the `addCourse` function:
```go
courses = addCourse(courses, "firstName", "lastName", d.Format("01/02/2006"))
```

Then, you can compile the program:
```bash
go build
```

You then need to create the `.env` file to match your configuration. The needed variables are:
- DATABASE_URL_GO: The URL to the database (e.g. `user:password@tcp(localhost:3306)/database`)
- URL: The URL to the EPSI schedule.

It is advised to let the program run every day using a cron job:
```bash
0 0 * * * /path/to/epsi-edt-scrapper
```
