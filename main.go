package main

import (
	"database/sql"

	_ "github.com/lib/pq"

	"fmt"
	"os"
	"strings"

)

var FieldTypes map[string]string

func main() {
	FieldTypes = map[string]string{
		"bigint":                   "int64",
		"integer":                  "int",
		"smallint":                 "int",
		"double precision":         "float64",
		"character varying":        "string",
		"character":                "string",
		"text":                     "string",
		"bytea":                    "[]byte",
		"date":                     "time.Time",
		"datetime":                 "time.Time",
		"timestamp":                "time.Time",
		"timestamp with time zone": "time.Time",
		"numeric":                  "float64",
		"decimal":                  "float64",
		"bit":                      "uint64",
		"boolean":                  "bool",
	}

	Init()
}

func check(e error) {
	if e != nil {
		panic(e)
	}
}

func Init() {
	db, err := sql.Open("postgres", "postgres://user@ip_host:port/database_name?sslmode=disable")
	check(err)

	qry, err := db.Query("SELECT table_name FROM information_schema.tables WHERE table_schema='public' AND table_type='BASE TABLE'")

	var (
		models    []string
		NameTable string
	)

	for qry.Next() {
		qry.Scan(&NameTable)
		models = append(models, NameTable)
	}

	for _, modelName := range models {
		res, err := db.Prepare("SELECT column_name, data_type, is_nullable, column_default FROM information_schema.columns WHERE table_name ='" + modelName + "'")
		check(err)

		qry, err := res.Query()
		check(err)

		f, err := os.Create("models/" + modelName + ".go")
		check(err)

		addImport(f)

		modelName := strings.Replace(modelName, "_", " ", -1)
		modelName = strings.Title(modelName)
		modelName = strings.ToUpper(string(modelName[0])) + modelName[1:]
		modelName = strings.Replace(modelName, " ", "", -1)
		addStruct(f, modelName, qry)
		addCRUD(f, modelName)

		addFetchOne(f, modelName)
		addFetchAll(f, modelName)
		addCreate(f, modelName)
		addUpdate(f, modelName)
		addRemove(f, modelName)

		res.Close()
		qry.Close()
		f.Close()
	}

	defer db.Close()

}

func addCRUD(f *os.File, modelName string) {
	modelNameFormat := strings.ToLower(string(modelName[0])) + modelName[1:]

	f.WriteString(
		`func ` + modelName + `CRUD(crud *gin.RouterGroup) {
		` + modelNameFormat + `:= crud.Group("/` + modelNameFormat + `")
		{
			` + modelNameFormat + `.GET("/:id", ` + modelName + `FetchOne)
			` + modelNameFormat + `.GET("/", ` + modelName + `FetchAll)
			` + modelNameFormat + `.POST("/", ` + modelName + `Create)
			` + modelNameFormat + `.PUT("/:id", ` + modelName + `Update)
			` + modelNameFormat + `.DELETE("/:id", ` + modelName + `Remove)
		}
	}
	`)
}

func addFetchOne(f *os.File, modelName string) {
	singularModel := strings.ToLower(string(modelName[0])) + modelName[1:len(modelName)-1]
	pluralModel := strings.ToLower(string(modelName[0])) + modelName[1:len(modelName)]

	f.WriteString(`
	// @Title get`+modelName+`
	// @Description retrieves `+singularModel+` by ID
	// @Accept  json
	// @Tags `+pluralModel+`
	// @Param   id     path    int     true        "`+singularModel+` ID"
	// @Success 200 {array}  models.`+modelName+`
	// @Failure 400 {string} code   "`+singularModel+` ID must be specified"
	// @Resource /`+pluralModel+`
	// @Router /`+pluralModel+`/{id} [get]
	func ` + modelName + `FetchOne(c * gin.Context){
		id := c.Param("id")
		db, err := db.Database()
		defer db.Close()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
		} else {
			var ` + singularModel + ` ` + modelName + `
			if err := db.Find(&` + singularModel + `, id).Error; err != nil {
				c.JSON(http.StatusNotFound, gin.H{"message": err.Error()})
			} else {
				c.JSON(http.StatusOK, ` + singularModel + `)
			}
		}
	}`)
}

func addFetchAll(f *os.File, modelName string) {
	pluralModel := strings.ToLower(string(modelName[0])) + modelName[1:len(modelName)]

	f.WriteString(`
	// @Title get`+modelName+`
	// @Description retrieves a list of `+pluralModel+` 
	// @Accept  json
	// @Tags `+pluralModel+`
	// @Success 200 {array}  models.`+modelName+`
	// @Resource /`+pluralModel+`
	// @Router /`+pluralModel+`/[get]
	func ` + modelName + `FetchAll(c *gin.Context) {
		db, err := db.Database()
		defer db.Close()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
		} else {
			id := c.Params.ByName("id")
			if err := db.Where("id = ?", id).First(&` + pluralModel + `).Error; err != nil {
				c.JSON(http.StatusNotFound, gin.H{"message": err.Error()})
			} else {
				c.JSON(http.StatusOK, ` + pluralModel + `)
			}
		}
	}`)
}

func addCreate(f *os.File, modelName string) {
	singularModel := strings.ToLower(string(modelName[0])) + modelName[1:len(modelName)-1]
	pluralModel := strings.ToLower(string(modelName[0])) + modelName[1:len(modelName)]

	f.WriteString(`
	// @Title get`+modelName+`
	// @Description retrieves `+singularModel+` by ID
	// @Accept  json
	// @Tags `+modelName+`
	// @Success 200 {array} models.`+modelName+`
	// @Failure 400 {string} code   "`+singularModel+` ID must be specified"
	// @Resource /`+pluralModel+`
	// @Router /`+pluralModel+`/ [post]
	func ` + modelName + `Create(c *gin.Context) {
		var ` + singularModel + ` ` + modelName + `
		if err := c.BindJSON(&` + singularModel + `); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
		} else {
			db, err := db.Database()
			defer db.Close()
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
			} else {
				if err := db.Create(&` + singularModel + `).Error; err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
				} else {
					c.JSON(http.StatusCreated, ` + singularModel + `)
				}
			}
		}
		return
	}`)
}

func addUpdate(f *os.File, modelName string) {
	singularModel := strings.ToLower(string(modelName[0])) + modelName[1:len(modelName)-1]
	pluralModel := strings.ToLower(string(modelName[0])) + modelName[1:len(modelName)]

	f.WriteString(`
	// @Title update`+modelName+`
	// @Description retrieves `+singularModel+` by ID
	// @Accept  json
	// @Param   id     path    int     true        "`+singularModel+` ID"
	// @Tags `+pluralModel+`
	// @Success 200 {array} models.`+modelName+`
	// @Failure 400 {string}  code   "`+singularModel+` ID must be specified"
	// @Resource /`+pluralModel+`
	// @Router /`+pluralModel+`/{id} [put]
	func ` + modelName + `Update(c *gin.Context) {
		var ` + singularModel + ` ` + modelName + `
		id := c.Params.ByName("id")
		db, err := db.Database()
		defer db.Close()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
		} else {
			if err := db.Where("id = ?", id).First(&` + singularModel + `).Error; err != nil {
				c.JSON(http.StatusNotFound, gin.H{"message": err.Error()})
			} else {
				if e := c.BindJSON(&` + singularModel + `); e != nil {
					c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
				} else {
					db.Save(&` + singularModel + `)
					c.JSON(http.StatusOK, ` + singularModel + `)
				}
			}
		}
	}`)
}

func addRemove(f *os.File, modelName string) {
	singularModel := strings.ToLower(string(modelName[0])) + modelName[1:len(modelName)-1]
	pluralModel := strings.ToLower(string(modelName[0])) + modelName[1:len(modelName)]

	f.WriteString(`
	// @Title remove`+modelName+`
	// @Description retrieves `+singularModel+` by ID
	// @Param   id     path    int     true        "`+singularModel+` ID"
	// @Accept  json
	// @Tags `+pluralModel+`
	// @Success 200 {array} models.`+modelName+`
	// @Failure 400 {string} code   "`+singularModel+` ID must be specified"
	// @Resource /`+pluralModel+`
	// @Router /`+pluralModel+`/{id} [delete]
	func ` + modelName + `Remove(c *gin.Context) {
		var ` + singularModel + ` ` + modelName + `
		db, err := db.Database()
		defer db.Close()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
		} else {
			id := c.Params.ByName("id")
			if err := db.Where("id = ?", id).First(&` + singularModel + `).Error; err != nil {
				c.JSON(http.StatusNotFound, gin.H{"message": err.Error()})
			} else {
				db.Delete(&` + singularModel + `)
				c.JSON(http.StatusOK, ` + singularModel + `)
			}
		}
	}`)

}
func addImport(f *os.File) {

	f.WriteString(`package models
		import (
				
			"net/http"
			"github.com/gin-gonic/gin"
		)
		`)

}

func addStruct(f *os.File, modelName string, qry *sql.Rows) {

	f.WriteString(`
type ` + modelName + ` struct {
`)
	var (
		ColumnName    string
		DataType      string
		IsNullable    string
		ColumnDefault string
	)
	for qry.Next() {
		qry.Scan(&ColumnName, &DataType, &IsNullable, &ColumnDefault)
		Title := strings.Title(ColumnName)
		i := strings.Index(DataType, "(")
		if i != -1 {
			DataType = DataType[0:i]
		}

		fmt.Printf("Column=%s, Type=%s, Null=%s, DefaultValue=%s\n", ColumnName, DataType, IsNullable, ColumnDefault)
		name := Title
		tp := FieldTypes[DataType]
		sql := "`gorm:\"column:" + ColumnName + ";"
		if IsNullable == "NO" {
			sql += "not null;"
		}
		sql += `"`

		sql += " json:\"" + ColumnName + "\"`"

		line := fmt.Sprintf("  %-10s\t%-10s\t%-20s", name, tp, sql)
		f.WriteString(line + "\n")
		ColumnName, DataType, IsNullable, ColumnDefault = "", "", "", ""
	}
	f.WriteString("}\n")

}