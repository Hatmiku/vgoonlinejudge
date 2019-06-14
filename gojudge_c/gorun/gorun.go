package main
/*
#include "gojudge_c.h"
int compile(char* langunage,char* filename);
int execute_test_program(char* filename,int as_size,int cpu_time);
int check_answer(char* file1,char* file2);
int filesize(FILE* file);
int test_exec(char *filename);
*/
import "C"
import (
	"net"
	"strconv"
	"message/protofile"
	"github.com/golang/protobuf/proto"
	"gojudge_c"
	"fmt"
	"io/ioutil"
	"bytes"
	"encoding/binary"
)
type judgeRequest struct{
	problemid int32
	usercode string
}

type judgeQueue struct{
	queuelen int32
	//判题队列长度
	splitrequest []judgeRequest
	//判题切片
}

//用来传递channel 的结构
type chanPage struct{
	//每一个chan 对应一个判题环境
	ch0  chan int
	ch1  chan int
	ch2  chan int
	ch3  chan int
}

func (J *judgeQueue) front()(judgeRequest,bool){
	frequest :=judgeRequest{}
	if len (J.splitrequest)>=1 {
		frequest = J.splitrequest[0]
		J.splitrequest = J.splitrequest[1:]
		J.queuelen = J.queuelen-1
		return frequest,true
	}
	return frequest,false
}

//int类型转换为byte数组的功能函数
func IntToBytes(n int)[]byte{
	x := int32(n)
	bytesBuffer := bytes.NewBuffer([]byte{})
	binary.Write(bytesBuffer,binary.BigEndian,x)
	return bytesBuffer.Bytes()
}

//byte数组转换为int类型的功能函数
func BytesToInt(b []byte)int{
	bytesBuffer := bytes.NewBuffer(b)
	var x int32
	binary.Read(bytesBuffer,binary.BigEndian,&x)
	return int(x)
}

//读取包内容
func readPacket(conn *net.Conn)(bool,protofile.Packet){
	var p protofile.Packet;
	head := make ([]byte,4)//四个字节的包长度

	_,err := (*conn).Read(head)//先读取四个字节
	if err!=nil{
		fmt.Println("error in read from conn to [] head")
		return false,p
	}
	ihead := BytesToInt(head)//将读取到的四个字节转换为int32类型
	if ihead <0 || ihead > 65535{
		return false,p
	}
	fmt.Println("body length is :",ihead)
	//根据包长度读取剩余内容
	body := make([]byte,ihead)
	_,err = (*conn).Read(body)
	if err!=nil{
		fmt.Printf("error is %v",err)
		return false,p
	}
	err = proto.Unmarshal(body,&p)
	if err!=nil{
		fmt.Println("error to read packet from client")
	}
	return true,p
}

//回送信息
func sendMessage(message []byte,conn* net.Conn)bool{
	meslen := len(message)
	phead  := IntToBytes(meslen)
	(*conn).Write(phead)
	(*conn).Write(message)
	return true
}

//处理链接
func handelConn(conn *net.Conn,ch0,ch1,ch2,ch3 chan int)bool{
	defer (*conn).Close()
	flag,pack := readPacket(conn)
	if !flag{
		return false
	}
	packcmd := pack.GetCommand()
	switch packcmd{
		//判题请求
		case protofile.EnumMessageCommand_enumJudgeRequest:{
			handelJudge(pack,ch0,ch1,ch2,ch3,conn)

			//因为handelJudge 在里面向channel 写信息 所以阻塞了
			//发回客户端结果的时候 要在阻塞处理之前
			//获取结果
			//发回客户端结果
			/*
			judgeresponse:=protofile.JudgeResponse{}
			rpack := protofile.Packet{}

			judgeresponse.JudgeSol= judgesol
			rpack.Version = 1
			rpack.Command = protofile.EnumMessageCommand_enumJudgeRresponse

			jdata,err := proto.Marshal(&judgeresponse)
			if err!=nil{
				fmt.Println(err)
			}
			rpack.Serialized = jdata
			data,err := proto.Marshal(&rpack)
			if err!=nil{
				fmt.Println(err)
			}
			fmt.Println("sendMessage to client");
			sendMessage(data,conn)
			*/
		}
		//查询请求
		//根据id查询
		case protofile.EnumMessageCommand_enumQueryRequest:{
			//解包
			qrequest:=protofile.QueryRequest{}
			qdata:=pack.Serialized

			err:=proto.Unmarshal(qdata,&qrequest)
			if err!=nil{
				fmt.Println("Error at Unmarshal request message");
				return false;
			}
			helper :=gojudge_c.LocaldbHelper{}
			sqlstruc,flag :=helper.InitConfig()

			fmt.Println(flag)
			fmt.Println(sqlstruc)

			if helper.InitConnection("local_judge")==false{
				fmt.Println("Sql connection error")
				return false
			}

			if helper.Connection == nil{
				fmt.Println("Connection is nil")
				return false
			}
			problem,flag:=helper.GetProblemByID(qrequest.ProblemId);
			fmt.Println(problem);
			//打包发给客户端
			/*	
			rpack.Version = 1
			rpack.Command = protofile.EnumMessageCommand_enumJudgeRresponse

			jdata,err := proto.Marshal(&judgeresponse)
			if err!=nil{
				fmt.Println(err)
			}
			rpack.Serialized = jdata
			data,err := proto.Marshal(&rpack)
			if err!=nil{
				fmt.Println(err)
			}
			fmt.Println("sendMessage to client");
			sendMessage(data,conn)
			*/
			/*
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
			*/

			rpack:=protofile.Packet{}
			rpack.Version=1
			rpack.Command=protofile.EnumMessageCommand_enumQueryResponse
			qresponse:=protofile.QueryResponse{}

			qresponse.ProblemId = problem.Id
			qresponse.ProblemTitle = problem.Title
			qresponse.ProblemDes = problem.Des
			qresponse.ProblemSampleIn = problem.Sample_in
			qresponse.ProblemSampleOut = problem.Sample_out
			qresponse.ProblemTime = problem.Time
			qresponse.ProblemMem = problem.Mem

			rdata,err:= proto.Marshal(&qresponse)
			if err!=nil{
				fmt.Println("err at Marshal qresponse")
				return false
			}
			rpack.Serialized = rdata
			data,err:= proto.Marshal(&rpack)
			if err!=nil{
				fmt.Println("err at Marshal qresponse")
				return false
			}
			sendMessage(data,conn)
		}
		case protofile.EnumMessageCommand_enumQueryAllRequest:{

			helper :=gojudge_c.LocaldbHelper{}
			sqlstruc,flag :=helper.InitConfig()

			fmt.Println(flag)
			fmt.Println(sqlstruc)

			if helper.InitConnection("local_judge")==false{
				fmt.Println("Sql connection error")
				return false
			}

			if helper.Connection == nil{
				fmt.Println("Connection is nil")
				return false
			}
			problemlist,err:=helper.GetProblemTitleList();
			if err==false{
				fmt.Println("error at get problem")
				return false
			}
			//helper.GetAllProblem();
			//problem,flag:=helper.GetProblemByID(qrequest.ProblemId);
			fmt.Println(problemlist);//测试输出
			//然后在这里给客户端回包

			//回发给客户端的结果
			raresponse:=protofile.QueryAllResponse{}

			rpack:= protofile.Packet{}
			rpack.Command = protofile.EnumMessageCommand_enumQueryAllResponse

			rproblemlist:=raresponse.GetProblemList()
			var qresponse *protofile.QueryResponse

			for i:=0;i<len(problemlist);i++{
				qresponse = new(protofile.QueryResponse)
				(*qresponse).ProblemId = problemlist[i].Id
				(*qresponse).ProblemTitle = problemlist[i].Title
				rproblemlist = append(rproblemlist,qresponse)
			}
			raresponse.ProblemList = rproblemlist

			radata,nerr:=proto.Marshal(&raresponse)
			if nerr!=nil{
				fmt.Println("error at marshal r all response")
			}
			rpack.Serialized = radata
			data,nerr:=proto.Marshal(&rpack)
			if nerr!=nil{
				fmt.Println("error at marshal rpack q all response")
			}
			sendMessage(data,conn)
		}
	}
	return true;
}

//用来回发信息的包装函数
func rebackJudge(conn *net.Conn,judgeSol int32)bool{
	judgeresponse:=protofile.JudgeResponse{}
	rpack := protofile.Packet{}

	judgeresponse.JudgeSol= judgeSol
	rpack.Version = 1
	rpack.Command = protofile.EnumMessageCommand_enumJudgeRresponse

	jdata,err := proto.Marshal(&judgeresponse)
	if err!=nil{
		fmt.Println(err)
		return false;
	}
	rpack.Serialized = jdata
	data,err := proto.Marshal(&rpack)
	if err!=nil{
		fmt.Println(err)
		return false;
	}
	fmt.Println("sendMessage to client");
	return sendMessage(data,conn)
}

func handelJudge(pack protofile.Packet,ch0,ch1,ch2,ch3 chan int,conn *net.Conn)int32{
	logdata := pack.Serialized
	loginfo := protofile.JudgeRequest{}
	err:=proto.Unmarshal(logdata,&loginfo)
	if err!=nil{
		fmt.Println("error to read log info from packet")
	}
	//获取题目id和代码String
	usercode := loginfo.GetUserCode()
	problemid := loginfo.GetProblemId()
	//先打印一下信息

	fmt.Println("judge request from client ")
	fmt.Println("problem id :%d",problemid)
	fmt.Println("user code : %s",usercode)
	var s gojudge_c.Sqlconfstruc
	fmt.Println(s)
	var dirpath string
	var judgesol int32
	select{
		//等待channel 信息
		case <-ch0:{
			dirpath = "./thread"+strconv.Itoa(0)
			fmt.Println(dirpath)
			//do some thing
			p,_:=initJudgeEnv(dirpath,problemid,&usercode)
			judgesol=judgeCode(dirpath,p.Mem,p.Time)
			rebackJudge(conn,judgesol)
			//释放资源
			//会在这里阻塞 直到channnel 中资源被取走
			//所以在阻塞之前reback结果
			ch0<-0
		}
		case <-ch1:{
			dirpath = "./thread"+strconv.Itoa(1)
			fmt.Println(dirpath)
			//do some thing
			p,_:=initJudgeEnv(dirpath,problemid,&usercode)
			judgesol=judgeCode(dirpath,p.Mem,p.Time)
			rebackJudge(conn,judgesol)
			ch1<-1
		}
		case <-ch2:{
			dirpath = "./thread"+strconv.Itoa(2)
			fmt.Println(dirpath)
			p,_:=initJudgeEnv(dirpath,problemid,&usercode)
			judgesol=judgeCode(dirpath,p.Mem,p.Time)
			rebackJudge(conn,judgesol)
			ch2<-2
		}
		case <-ch3:{
			dirpath = "./thread"+strconv.Itoa(3)
			fmt.Println(dirpath)
			p,_:=initJudgeEnv(dirpath,problemid,&usercode)
			judgesol=judgeCode(dirpath,p.Mem,p.Time)
			rebackJudge(conn,judgesol)
			ch3<-3
		}
	}
	return judgesol
}
//初始化判题环境 下载题目数据 保存用户代码
//这里想在根目录生成四个文件夹 每个文件夹代表一个题目环境 锁起来
func initJudgeEnv(path string,problemid int32,usercode *string)(gojudge_c.Problemlist,bool){
	fmt.Println("Write path is :"+path+"/test_program.c")
	problem := gojudge_c.Problemlist{}
	flag:=false
	helper :=gojudge_c.LocaldbHelper{}
	//defer helper.Connection.Close()
	sqlstruc,flag :=helper.InitConfig()
	fmt.Println(flag)
	fmt.Println(sqlstruc)
	//保存用户代码到本地
	ioutil.WriteFile(path+"/test_program.c",[]byte(*usercode),0644)

	//从根据problemid 从数据库获取题目信息
	if helper.InitConnection("local_judge")==false{
		fmt.Println("Sql connection error")
		return problem,false
	}

	if helper.Connection == nil{
		fmt.Println("Connection is nil")
		return problem,false
	}
	problem,flag =helper.GetProblemByID(problemid)

	if flag==false{
		fmt.Println("Sql Error")
		return problem,false
	}
	fmt.Println(problem)
	//保存problem_in problem_out
	ioutil.WriteFile(path+"/problem.in",[]byte(problem.In),0644)
	ioutil.WriteFile(path+"/problem.out",[]byte(problem.Out),0644)
	return problem,true;
}


//判题C代码封装
//AC 0
//WA 1
//CE 2
//RE 7
//TLE 14
//SYSTEMCALL 21
func judgeCode(path string,mem int32,time int32)int32{
	fmt.Println(path+"/test_program.c")
	p := C.compile(C.CString("C"),C.CString(path));
	var exe_flag C.int
	exe_flag = C.int(1)
	if p!=0{
		fmt.Println("CE");
		return 2;
	}else{
		exe_flag = C.execute_test_program(C.CString(path),C.int(mem),C.int(time));
		fmt.Println("exe_flag:%d\n",exe_flag);
		if exe_flag==C.int(0){
			check_flag := C.check_answer(C.CString(path+"/test_program.out"),C.CString(path+"/problem.out"))
			if check_flag==C.int(0){
				fmt.Println("AC")
				return 0;
			}else{
				fmt.Println("WA")
				return 1;
			}
		}else if exe_flag==C.int(14){
			fmt.Println("TLE")
		}else if exe_flag==C.int(7){
			fmt.Println("RE")
		}else if exe_flag==C.int(21){
			fmt.Println("SYSTEMCALL")
		}
	}
	return int32(exe_flag)
}

func initChanPage(ch0,ch1,ch2,ch3 chan int){
	ch0 <-0
	ch1 <-1
	ch2 <-2
	ch3 <-3
}

func main(){
	//初始化chanpage

	ch0 := make(chan int)
	ch1 := make(chan int)
	ch2 := make(chan int)
	ch3 := make(chan int)

	go initChanPage(ch0,ch1,ch2,ch3)

	Listener,err := net.Listen("tcp","127.0.0.1:9998")
	if err!=nil{
		fmt.Println(err)
	}
	for{
		conn,err := Listener.Accept()
		if err!=nil{
			fmt.Println("err to accept from listener")
			continue
		}
		go handelConn(&conn,ch0,ch1,ch2,ch3)
	}
	defer Listener.Close()

}
