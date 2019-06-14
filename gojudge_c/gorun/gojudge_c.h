#ifndef GOJUDGE_C_H
#define GOJUDGE_C_H

#include<errno.h>
#include<unistd.h>
#include<stdio.h>
#include<stdlib.h>
#include<string.h>
#include<sys/types.h>
#include<sys/wait.h>
#include<sys/user.h>
#include<sys/time.h>
#include<sys/resource.h>
#include<sys/ptrace.h>
#include<sys/reg.h>


#define length 128
#define code_length 1024
#define answer_length 102400



int get_split(char* str);//分割符':'处理
int compile(char* langunage,char* filename);//用来处理编译的函数 成功返回0 编译失败返回1
int execute_test_program(char* filename,int as_size,int cpu_time);//执行提交代码函数  在限制内返回0 TLE 返回14 RE 返回7 禁止系统调用返回 21
int check_answer(char* file1,char* file2);//比对结果 结果正确返回0 结果不正确返回1
int filesize(FILE* file);//获取文件字节数
int test_exec(char *filename);

#endif



