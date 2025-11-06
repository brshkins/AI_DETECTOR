@echo off
REM ============================================
REM Скрипт для запуска Python ML Service на Windows
REM ============================================

echo.
echo ========================================
echo Starting Drowsiness Detection ML Service
echo ========================================
echo.

REM Перейти в папку ml_service
cd /d "%~dp0ml-model\ml_service"

if not exist venv (
    echo  Creating virtual environment...
    python -m venv venv
    echo  Virtual environment created
    echo.
)

echo Activating virtual environment...
call venv\Scripts\activate.bat

echo.
echo  Virtual environment activated
echo.

REM Проверим, нужно ли установить зависимости
if not exist "venv\Lib\site-packages\torch" (
    echo    Installing dependencies (first time only)...
    echo    This may take 5-10 minutes...
    echo.
    pip install --upgrade pip
    pip install torch torchvision pillow opencv-python grpcio grpcio-tools
    echo Dependencies installed
    echo.
)

REM Проверим, нужно ли создать proto файлы
if not exist "drowsiness_pb2.py" (
    echo Generating proto files...
    if not exist "drowsiness.proto" (
        echo  Copying drowsiness.proto...
        copy "..\..\go-backend\pkg\pb\drowsiness.proto" . >nul 2>&1
    )
    python -m grpc_tools.protoc -I. --python_out=. --grpc_python_out=. drowsiness.proto
    echo Proto files generated
    echo.
)

REM Запускаем сервис
echo ========================================
echo 🚀 Running ML Service...
echo ========================================
echo.

python drowsiness_grpc_service.py

pause
