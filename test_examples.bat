@echo off
chcp 65001 >nul

REM LLM Mock API Test Script (Windows)
REM Make sure server is running before use: go run .

set BASE_URL=http://localhost:8080

echo ======================================
echo LLM Mock API Test Script
echo ======================================
echo.

:menu
echo Select test to run:
echo   1) Health Check
echo   2) List Models
echo   3) Simple Chat
echo   4) Streaming Response
echo   5) Reasoning Mode
echo   6) Basic Chat Chain
echo   7) Complex Chat Chain
echo   8) Tool Calls
echo   9) List Test Tools
echo   10) Invoke Test Tool
echo   11) Delay Fault
echo   12) Slow Stream
echo   13) Concurrent Tool Calls
echo   14) Fault Presets
echo   0) Run All Tests
echo   q) Quit
echo.

set /p choice="Enter option: "

if "%choice%"=="1" goto health
if "%choice%"=="2" goto models
if "%choice%"=="3" goto simple
if "%choice%"=="4" goto streaming
if "%choice%"=="5" goto reasoning
if "%choice%"=="6" goto chain_basic
if "%choice%"=="7" goto chain_complex
if "%choice%"=="8" goto tool_calls
if "%choice%"=="9" goto list_tools
if "%choice%"=="10" goto invoke_tool
if "%choice%"=="11" goto fault_delay
if "%choice%"=="12" goto slow_stream
if "%choice%"=="13" goto concurrent_tools
if "%choice%"=="14" goto fault_presets
if "%choice%"=="0" goto all
if "%choice%"=="q" goto quit
if "%choice%"=="Q" goto quit
echo Invalid option
echo.
goto menu

:health
echo [Test 1] Health Check
curl -s %BASE_URL%/health
echo.
echo.

goto menu

:models
echo [Test 2] List Models
curl -s %BASE_URL%/api/v1/models
echo.
echo.

goto menu

:simple
echo [Test 3] Simple Chat
curl -s -X POST "%BASE_URL%/api/v1/chat/completions" -H "Content-Type: application/json" -d "{\"messages\": [{\"role\": \"user\", \"content\": \"Hello\"}]}"
echo.
echo.

goto menu

:streaming
echo [Test 4] Streaming Response
curl -s -X POST "%BASE_URL%/api/v1/chat/completions" -H "Content-Type: application/json" -d "{\"messages\": [{\"role\": \"user\", \"content\": \"Hello\"}], \"stream\": true}"
echo.
echo.

goto menu

:reasoning
echo [Test 5] Reasoning Mode
curl -s -X POST "%BASE_URL%/api/v1/chat/completions" -H "Content-Type: application/json" -d "{\"messages\": [{\"role\": \"user\", \"content\": \"2+2=?\"}], \"reasoning\": {\"effort\": \"high\"}, \"stream\": true}"
echo.
echo.

goto menu

:chain_basic
echo [Test 6] Basic Chat Chain
curl -s -X POST "%BASE_URL%/api/v1/chat/completions" -H "Content-Type: application/json" -d "{\"messages\": [{\"role\": \"system\", \"content\": \"#CHAIN: reasoning-content\"}, {\"role\": \"user\", \"content\": \"Hello\"}], \"stream\": true}"
echo.
echo.

goto menu

:chain_complex
echo [Test 7] Complex Chat Chain
curl -s -X POST "%BASE_URL%/api/v1/chat/completions" -H "Content-Type: application/json" -d "{\"messages\": [{\"role\": \"system\", \"content\": \"#CHAIN: reasoning{text=First step}-content{text=Result 1}-reasoning{text=Second step}-content{text=Result 2}\"}, {\"role\": \"user\", \"content\": \"Tell me a story\"}], \"stream\": true}"
echo.
echo.

goto menu

:tool_calls
echo [Test 8] Tool Calls
curl -s -X POST "%BASE_URL%/api/v1/chat/completions" -H "Content-Type: application/json" -d "{\"messages\": [{\"role\": \"system\", \"content\": \"#CHAIN: content-tool_calls\"}, {\"role\": \"user\", \"content\": \"What's the weather?\"}], \"stream\": true, \"tools\": [{\"type\": \"function\", \"function\": {\"name\": \"get_weather\", \"description\": \"Get weather\"}}]}"
echo.
echo.

goto menu

:list_tools
echo [Test 9] List Test Tools
curl -s %BASE_URL%/test-tools
echo.
echo.

goto menu

:invoke_tool
echo [Test 10] Invoke Test Tool
curl -s -X POST "%BASE_URL%/test-tools/get_weather/invoke" -H "Content-Type: application/json" -d "{\"location\": \"Beijing\", \"unit\": \"celsius\"}"
echo.
echo.

goto menu

:fault_delay
echo [Test 11] Delay Fault
curl -s -X POST "%BASE_URL%/api/v1/chat/completions" -H "Content-Type: application/json" -d "{\"messages\": [{\"role\": \"system\", \"content\": \"#CHAIN: content{fault=delay,fault_duration=1s,text=Delayed message}\"}, {\"role\": \"user\", \"content\": \"Hello\"}], \"stream\": true}"
echo.
echo.

goto menu

:slow_stream
echo [Test 12] Slow Stream
curl -s -X POST "%BASE_URL%/api/v1/chat/completions" -H "Content-Type: application/json" -d "{\"messages\": [{\"role\": \"system\", \"content\": \"#CHAIN: content{char_delay=50ms,chunk_size=2,text=This is a slow stream}\"}, {\"role\": \"user\", \"content\": \"Hello\"}], \"stream\": true}"
echo.
echo.

goto menu

:concurrent_tools
echo [Test 13] Concurrent Tool Calls
curl -s -X POST "%BASE_URL%/api/v1/chat/completions" -H "Content-Type: application/json" -d "{\"messages\": [{\"role\": \"system\", \"content\": \"#CHAIN: reasoning, parallel:tool_calls, content\"}, {\"role\": \"user\", \"content\": \"Get weather and calculate\"}], \"stream\": true}"
echo.
echo.

goto menu

:fault_presets
echo [Test 14] Fault Presets
curl -s %BASE_URL%/fault-presets
echo.
echo.

goto menu

:all
echo [Run All Tests]
echo.
call :health
call :models
call :simple
call :streaming
call :reasoning
call :chain_basic
call :chain_complex
call :tool_calls
call :list_tools
call :invoke_tool
call :fault_delay
call :slow_stream
call :concurrent_tools
call :fault_presets
echo All tests completed!

goto menu

:quit
echo Exit
echo.
