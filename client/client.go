package main

import(
	"os"
	"net"
	"fmt"
	"log"
	"bytes"
	"strconv"
	"io/ioutil"
	"encoding/binary"
	"github.com/golang/protobuf/proto"
	"message/protofile"
)

func main(){
	const(
		EnumQueryByID = iota
		EnumQueryAll
		EnumJudgeCode
	)
	var Command int32 = -1
	var pid int32 = -1
	//os.Args 获取参数
	if len(os.Args)<2{
		fmt.Println("Args error...")
		return
	}
	if os.Args[1]=="-q"{
		//根据题目编号查询
		//后跟题目ID
		if len(os.Args)!=3{
			fmt.Println("Args error...")
			return
		}
		_pid,err:=strconv.Atoi(os.Args[2])
		if err!=nil{
			fmt.Println("Args error...")
			return
		}
		pid=int32(_pid)
		Command = EnumQueryByID
	}else if os.Args[1]=="-qa"{
		//查询所有
		if len(os.Args)!=2{
			fmt.Println("Args error...")
			return
		}
		Command = EnumQueryAll
	}else if os.Args[1]=="-j"{
		//提交代码
		//后跟文件名
		if len(os.Args)!=4{
			fmt.Println("Args error...")
			return
		}
		Command = EnumJudgeCode
	}else{
		fmt.Println("Args error...")
		return
	}

	//先连接服务器
	var addr string ="127.0.0.1:9998"
	conn,err := net.Dial("tcp",addr)
	if err!=nil{
		log.Printf("error net.Dial %v",err)
		return
	}
	p:= protofile.Packet{}
	p.Version = 1

	//根据参数执行不同的操作
	switch Command{
		case EnumQueryAll:{
			p.Command = protofile.EnumMessageCommand_enumQueryAllRequest

			var qrequest protofile.QueryRequest
			qrequest.ProblemId = 1
			qdata,err:=proto.Marshal(&qrequest)
			if err!=nil{
				fmt.Println("queryrequest error to marshal qrequest")
			}
			p.Serialized = qdata

			data,err := proto.Marshal(&p)
			if err!=nil{
				fmt.Println("queryall request error to marshal packet")
			}
			sendMessage(data,&conn)
		}
		case EnumQueryByID:{
			p.Command = protofile.EnumMessageCommand_enumQueryRequest
			var qrequest protofile.QueryRequest
			qrequest.ProblemId = pid
			qdata,err:=proto.Marshal(&qrequest)
			if err!=nil{
				fmt.Println("queryrequest error to marshal qrequest")
			}
			p.Serialized = qdata
			data,err := proto.Marshal(&p)

			if err!=nil{
				fmt.Println("queryrequest error to marshal packet")
			}
			sendMessage(data,&conn)
		}
		case EnumJudgeCode:{
			p.Command = protofile.EnumMessageCommand_enumJudgeRequest

			var jrequest protofile.JudgeRequest
			_pid,err:=strconv.Atoi(os.Args[3])
			if err!=nil{
				fmt.Println("Args error...!")
				return ;
			}
			pid=int32(_pid)
			jrequest.ProblemId =pid
			filecode,err:=ioutil.ReadFile(os.Args[2])
			if err!=nil{
				fmt.Println("open file error!")
			}
			jrequest.UserCode = string(filecode)
			/*
			jrequest.UserCode = `
#include<stdio.h>
#include<unistd.h>
int main(){
	int pid = fork();
	return 0;
}
`
			*/
			jdata,err:= proto.Marshal(&jrequest)
			if err!=nil{
				fmt.Println("judgerequest error to marshal jrequest")
			}

			p.Serialized = jdata
			data,err := proto.Marshal(&p)

			if err!=nil{
				fmt.Println("judgerequest error to marshal packet")
			}

			sendMessage(data,&conn)
		}
	}

	//等待服务器回包
	flag,pack:=readPacket(&conn);
	if flag==false {//处理失败
	}
	//获取pack id
	packcmd := pack.GetCommand()
	switch packcmd{
		//EnumMessageCommand_enumJudgeRequest   EnumMessageCommand = 0
		//判题结果
		case protofile.EnumMessageCommand_enumJudgeRresponse:{
			judgeresponse:=protofile.JudgeResponse{}
			logdata:=pack.Serialized
			err:=proto.Unmarshal(logdata,&judgeresponse)
			if err!=nil{
				fmt.Println("error to read message from packet")
			}
			switch judgeresponse.JudgeSol{
				case 0:{
					fmt.Println("AC")
				}
				case 1:{
					fmt.Println("WA")
				}
				case 2:{
					fmt.Println("CE")
				}
				case 7:{
					fmt.Println("RE")
				}
				case 14:{
					fmt.Println("TLE")
				}
				case 21:{
					fmt.Println("SYSTEMCALL")
				}
			}
		}
		//查询结果
		//EnumMessageCommand_enumQueryResponse    EnumMessageCommand = 3
		case protofile.EnumMessageCommand_enumQueryResponse:{
			qresponse:=protofile.QueryResponse{}
			logdata:=pack.Serialized
			err:=proto.Unmarshal(logdata,&qresponse)
			if err!=nil{
				fmt.Println("error to read message from paket")
			}
			fmt.Println(">>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>")
			fmt.Println("题目编号: ",qresponse.ProblemId)
			fmt.Println("标题: ")
			fmt.Println("	",qresponse.ProblemTitle)
			fmt.Println("题目描述: ")
			fmt.Println("	",qresponse.ProblemDes)
			fmt.Println("样例输入: ")
			fmt.Println("	",qresponse.ProblemSampleIn)
			fmt.Println("样例输出: ")
			fmt.Println("	",qresponse.ProblemSampleOut)
			fmt.Println("时间限制: ",qresponse.ProblemTime,"S","内存限制: ",qresponse.ProblemMem,"MB")
		}
		//获取所有题目列表
		case protofile.EnumMessageCommand_enumQueryAllResponse:{
			qaresponse:=protofile.QueryAllResponse{}
			logdata:=pack.Serialized
			err:=proto.Unmarshal(logdata,&qaresponse)
			if err!=nil{
				fmt.Println("err to read message from packet")
			}
			aproblemlist:=qaresponse.GetProblemList()
			for i:=0;i<len(aproblemlist);i++{
				fmt.Println("标题: ",(*aproblemlist[i]).ProblemTitle," ",(*aproblemlist[i]).ProblemId)
			}
		}
	}
	conn.Close()
}

func IntToBytes(n int)[]byte{
	x := int32(n)
	bytesBuffer := bytes.NewBuffer([]byte{})
	binary.Write(bytesBuffer,binary.BigEndian,x)
	return bytesBuffer.Bytes()
}

func BytesToInt(b []byte)int{
	bytesBuffer := bytes.NewBuffer(b)
	var x int32
	binary.Read(bytesBuffer,binary.BigEndian,&x)
	return int(x)
}

func sendMessage(message []byte,conn* net.Conn)bool{
	meslen := len(message)
	phead := IntToBytes(meslen)
	(*conn).Write(phead)
	(*conn).Write(message)
	return true
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
