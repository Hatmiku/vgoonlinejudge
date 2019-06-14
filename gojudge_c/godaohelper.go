package gojudge_c

import(
	"database/sql"
	_"github.com/Go-SQL-Driver/MySQL"
	"fmt"
	"encoding/xml"
	"io/ioutil"
	"os"
)

//题目信息表结构
type Problemlist struct{
	Id int32
	Title string
	Des string
	Sample_in string
	Sample_out string
	Time int32
	Mem int32
	In string
	Out string
}

//配置信息表结构
type Sqlconfstruc struct{
	XMLName xml.Name `xml:"sqlconn"`
	Sqlhost string `xml:"sqlhost"`
	Sqlport string `xml:"sqlport"`
	Sqluser string `xml:"sqluser"`
	Sqlpass string `xml:"sqlpass"`
}

//数据库链接 Helper
type LocaldbHelper struct{
	Sqltype string
	Sqlhost string
	Sqlport string
	Sqluser string
	Sqlpass string
	Connection *sql.DB
}

//加载配置文件
func (L* LocaldbHelper) InitConfig()(Sqlconfstruc,bool){
	s:= Sqlconfstruc{}
	cfile,err := os.Open("sqlconfig.xml")
	if err!=nil{
		return s,false
	}
	defer cfile.Close()

	data,err := ioutil.ReadAll(cfile)
	if err!=nil{
		fmt.Println("err at read xml file")
		return s,false
	}
	err = xml.Unmarshal(data,&s)
	if err!=nil{
		fmt.Println("err at read xml file in struct",err)
		return s,false
	}

	L.Sqltype = "mysql"

	L.Sqlhost = s.Sqlhost
	L.Sqlport = s.Sqlport
	L.Sqluser = s.Sqluser
	L.Sqlpass = s.Sqlpass
	return s,true
}

//初始化数据库链接
func (L* LocaldbHelper) InitConnection(tableName string)bool{
	db,err := sql.Open("mysql","root:@/"+tableName+"?charset=utf8")
	if err!=nil{
		fmt.Println(err)
		return false
	}
	L.Connection = db
	return true
}

//查询
func (L *LocaldbHelper) GetAllProblem()[]Problemlist{
	var ProblemSet []Problemlist
	return ProblemSet
}

//查询题目信息
func (L *LocaldbHelper)GetProblemByID(problemid int32)(Problemlist,bool){
	problem := Problemlist{}
	select_sql := "select * from problemlist where problem_id =?"
	select_err := L.Connection.QueryRow(select_sql,problemid).Scan(&problem.Id,&problem.Title,&problem.Des,&problem.Sample_in,&problem.Sample_out,&problem.Time,&problem.Mem,&problem.In,&problem.Out)
	if select_err!=nil{
		fmt.Println(select_err)
		return problem,false
	}
	return problem,true
}

//查询所有题目信息
func(L *LocaldbHelper)GetProblemTitleList()([]Problemlist,bool){
	var problemset []Problemlist
	select_sql :="select * from problemlist"
	select_rows,err:=L.Connection.Query(select_sql)
	if err!=nil{
		fmt.Println(err)
		return problemset,false
	}
	for select_rows.Next(){
		var problem Problemlist
		err := select_rows.Scan(&problem.Id,&problem.Title,&problem.Des,&problem.Sample_in,&problem.Sample_out,&problem.Time,&problem.Mem,&problem.In,&problem.Out)
		if err!=nil{
			return problemset,false
		}
		problemset=append(problemset,problem)
	}
	return problemset,true
}
