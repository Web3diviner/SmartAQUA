@echo off
REM Build script for Smart Fish Feeder SIL Simulation (Windows)

echo =========================================
echo Building SIL Simulation...
echo =========================================
echo.
echo Fixed: Using standalone simulation headers
echo No Arduino library dependencies required!
echo.

REM Clean and create build directory
if exist build (
    echo Cleaning old build directory...
    rmdir /s /q build
)
mkdir build
cd build

REM Run CMake
cmake ..

REM Build
cmake --build .

REM Check if build succeeded
if %ERRORLEVEL% EQU 0 (
    echo.
    echo =========================================
    echo Build successful!
    echo =========================================
    echo.
    echo Run simulation with: build\Debug\fishfeeder_sim.exe
    echo.
    echo The simulation will run 5 acceptance tests:
    echo   1. Dispense 50g with accuracy
    echo   2. Low feed blocking
    echo   3. Emergency stop on low DO
    echo   4. Timeout handling
    echo   5. Jam detection
    echo.
) else (
    echo.
    echo =========================================
    echo Build failed!
    echo =========================================
    echo.
    echo Please check the error messages above.
    echo If you see missing header errors, see MSVC_BUILD_FIX_APPLIED.md
    echo.
    exit /b 1
)
