@echo off
REM run.bat 30
start go run main.go server
for /L %%n in (1,1,%1) do start go run main.go client