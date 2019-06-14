#include "gojudge_c.h"



int get_split(char* str){//根据字符串头 保留信息 处理分割符 配置文件读取函数的工具函数
	char head[length];
	memset(head,0,sizeof(char)*length);
	int len = strlen(str);
	int flag = 0;
	int ret = 0;
	int pos = 0;
	for(int i=0;i<len;i++){
		if(str[i]==':')
			flag=1;
		if(!flag)
			head[i]=str[i];
		if(flag){//找到分割符
			//printf("%s\n",head);
			//1.先判断head 
			if(strcmp(head,"HOST")==0||strcmp(head,"host")==0)
				ret = 1;
			else if(strcmp(head,"DB")==0||strcmp(head,"db")==0)
				ret = 2;
			else if(strcmp(head,"PORT")==0||strcmp(head,"port")==0)
				ret = 3;
			else if(strcmp(head,"CHATSET")==0||strcmp(head,"charset")==0)
				ret = 4;
			else if(strcmp(head,"USER")==0||strcmp(head,"user")==0)
				ret = 5;
			else if(strcmp(head,"PASSWD")==0||strcmp(head,"passwd")==0)
				ret = 6;
			else 
				ret = -1;//配置文件格式错误
			//2.将原字符串处理
			for(int j=i+1;j<len-1;j++){//去掉\n
				str[pos]=str[j];
				pos++;		
			}
			str[pos]='\0';
			//printf("%s\n",str);
			break;
		}
	}
	return ret;
}



int compile(char* langunage,char* filename){//编译测试 编译成功返回0 失败返回1
		char filepath[50];
		strcat(filepath,"gcc ");
		strcat(filepath,filename);
		strcat(filepath,"/test_program.c ");
		strcat(filepath,"-o ");
		strcat(filepath,filename);
		strcat(filepath,"/test_program.o");
		int ret = system(filepath);
		if(ret==0)
			return 0;//编译成功
		else
			return 1;//编译失败
}



int execute_test_program(char* filename,int as_size,int cpu_time){//该函数用于运行编译成功的测试程序 并限制cpu运行时间和内存使用
	int syscall=0;
	int tle=0;
	int re=0;
	int exec_ret=0;
	int status;//用来获取子进程信息
	long rax;//用来存储 系统调用编号
	int alarm_flag;//用于闹钟是否成功设定的变量
	pid_t pid;
	struct rlimit r;//系统资源限制结构体
	pid=fork();
	if(pid==0){//子进程
		//这里如果是子进程 需要先ptrace 限制运行资源 然后使用exec函数加载
		//先ptrace
		//ptrace(PTRACE_TRACEME,0,NULL,NULL);
		//然后设置运行资源
		getrlimit(RLIMIT_AS,&r);
		r.rlim_cur=1024*1024*as_size;//XM 软限制
		r.rlim_max=1024*1024*as_size;//硬限制
		setrlimit(RLIMIT_AS,&r);
		printf("mem cur %d mem max %d\n",r.rlim_cur,r.rlim_max);

		getrlimit(RLIMIT_CPU,&r);
		printf("cpu cur %d cpu max %d\n",r.rlim_cur,r.rlim_max);
		r.rlim_cur=cpu_time;//1s软限制
		//r.rlim_max=cpu_time;//1s硬限制
		printf("cpu cur %d cpu max %d\n",r.rlim_cur,r.rlim_max);
		setrlimit(RLIMIT_CPU,&r);
		
		char execname[50];
		memset(execname,0,sizeof(execname));
		strcat(execname,filename);
		strcat(execname,"/test_program.o");
		printf("%s\n",execname);
		char filein[50];
		char fileout[50];
		memset(filein,0,sizeof(filein));
		memset(fileout,0,sizeof(fileout));
		strcat(filein,filename);
		strcat(filein,"/problem.in");

		strcat(fileout,filename);
		strcat(fileout,"/test_program.out");

		freopen(filein,"r",stdin);
		freopen(fileout,"w",stdout);

		//设置完成后 加载测试程序
		alarm_flag = alarm(1);	
		ptrace(PTRACE_TRACEME,0,NULL,NULL);
		
		exec_ret = execlp(execname,"test_program.o",NULL);

		if(exec_ret<0){//错误处理 exec调用失败
			perror("judge_core:execute_test_program:exec:");
			return 1;
		}
		//之后的处理交给父进程
	}else if(pid>0){//父进程
		while(1){//循环等待子进程信号
			waitpid(pid,&status,0);
			if(WIFEXITED(status)){//程序正常退出 而不是由信号引起的退出
				break;
			}
			if(WIFSIGNALED(status)){//这里,如果大佬是因为信号而停止,走这里
                        	//紧接着我们来判断是发送得什么信号
                        	//内存超限？还是时间超限？还是两者都？或者运行错误？
                        	//给大佬看病的时候到了
                       	 	if(WTERMSIG(status)==SIGXCPU||WTERMSIG(status)==SIGALRM){//时间超限
                             		tle=1;
					break;
                        	}else if(WTERMSIG(status)==SIGSEGV||WTERMSIG(status)==SIGKILL||WTERMSIG(status)==SIGABRT){//进程被杀死,SIGABRT是进程不确定态       
                                	re=1;
					break;
                        	}
				else{
                              		re=1;
					break;
                       		} 
               	 	}
			if(WIFSTOPPED(status)){//这里有个大坑 注意 是大坑
				//正常情况下 ptrace遇到了信号 就会抓过来 但是SIGKILL是不会被抓的
				//ptrace抓到信号之后干什么 暂停啊
				//暂停之后恢复子程序运行 看上起没什么毛病 嗯
				//问题大了
				//子程序信号被抓了 虽然知道有信号来过 但是就给父进程处理了 遇到SIGXCPU 不退出 对 就这么坑
				//本来SIGXCPU 这种信号的默认处理方式是直接退出程序的运行
				//但是ptrace(PTRACE_SYSCALL,pid,NULL,NULL) 这种信号就不处理了
				//而WTERMSIG 和 WIFSIGNALED 是子进程退出之后才执行的 因为子程序不会退出 所以应该硬WIFSTOPPED 和 WSTOPSIG来处理
				if(WSTOPSIG(status)==SIGXCPU){
					tle=1;
					break;
				}else if(WSTOPSIG(status)==SIGSEGV){
					re=1;
					break;
				}
			}	
			rax = ptrace(PTRACE_PEEKUSER,pid,ORIG_RAX*8,NULL);
			printf("rax rax rax rax :%ld\n",rax);
			//获取系统调用
			//发现程序使用禁止使用的系统调用
			//终止子进程
			if(rax==57||rax==56){
				syscall=1;
				ptrace(PTRACE_KILL,pid,NULL,NULL);
			}
			ptrace(PTRACE_SYSCALL,pid,NULL,NULL);
			
		}
		printf("syscall %d tle %d re %d\n",syscall,tle,re);
		if(syscall)
			return 21;//使用了禁止的系统调用
		else if(tle)
			return 14;//时间超限
		else if(re)
			return 7;//运行时错误
		else
			return 0;//程序正常运行出结果
		
	}else{//错误处理
		perror("judge_core:execute_test_program:fork:");
		return -1;
	}
}



int check_answer(char* file1,char* file2){//用于比对测试代码输出和标程输出结果
	FILE* f1=NULL;//以file1 为标准
	FILE* f2=NULL;
	int filelong1 =0;
	int filelong2=0;
	int flag=0;
	char buf1[answer_length];
	char buf2[answer_length];
	int file1c;
	int file2c;

	f1=fopen(file1,"r");
	f2=fopen(file2,"r");
	filelong1=filesize(f1);
	filelong2=filesize(f2);
	printf("file size:%d %d\n",filelong1,filelong2);
	if(filelong1!=filelong2){
		fclose(f1);
		fclose(f2);
		return 1;
	}
	if(filelong1==filelong2){//两文件字节数相等
		for(int i=0;i<filelong1;i++){
			file1c=fgetc(f1);
			file2c=fgetc(f2);
			printf("%c %c\n",file1c,file2c);
			if(file1c!=file2c){
				flag=1;
				break;
			}
		}
	}
	fclose(f1);
	fclose(f2);
	return flag;	
}



int filesize(FILE* file){//获取文件字节数的工具函数
	int size=0;
	fpos_t fpos;//当前位置
	fgetpos(file,&fpos);
	fseek(file,0,SEEK_END);
	size=ftell(file);
	fsetpos(file,&fpos);
	printf("%d\n",size);
	return size;
}


/*
int main(int argc,char* argv[]){
	int testprogram_flag=0;
	int problem_flag=0;
	//char *a[length];
	int exe_flag;
	
	
	int p = compile("C","test_program.c");
	printf("%d\n",p);
	if(p){//编译失败
		printf("CE\n");
	}else{

		//testprogram_flag = save_testprogram(HOST,USER,PASSWD,PORT,DB,"1");
		//problem_flag=save_pproblem(HOST,USER,PASSWD,DB,PORT,"1",&as_size,&cpu_time);
		//problem_flag = save_pproblem(HOST,USER,PASSWD,PORT,DB,"1");
		printf("testprogram_flag = %d problem_flag = %d \n",testprogram_flag,problem_flag);
		printf("as_size %d cpu_time %d\n",as_size,cpu_time);
		exe_flag=execute_test_program("./test_program.o",32,1);
		printf("exe_flag:%d\n",exe_flag);
		if(exe_flag==0){//运行成功 比对结果
			if(check_answer("./test_program.out","./problem.out")==0)
				printf("AC\n");
			else 
				printf("WA\n");
		}else if(exe_flag==14){
			printf("TLE\n");
		}else if(exe_flag==7){
			printf("RE\n");
		}else if(exe_flag==21){
			printf("SYSCALL\n");
		}
	}
	
	return 0;	
}
*/
