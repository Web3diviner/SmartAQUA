@echo off
echo ============================================================================
echo Smart Fish Feeder - LTspice Simulation Suite
echo Running all 6 scenarios...
echo ============================================================================
echo.

set LTSPICE="C:\Program Files\LTC\LTspiceXVII\XVIIx64.exe"

echo [1/6] Running Scenario 1: Sunny 24hr (~30 seconds)
%LTSPICE% -b scenario_sunny_24hr.cir
if errorlevel 1 (
    echo ERROR: Scenario 1 failed!
    pause
    exit /b 1
)
echo [1/6] Complete!
echo.

echo [2/6] Running Scenario 2: Cloudy 24hr (~30 seconds)
%LTSPICE% -b scenario_cloudy_24hr.cir
if errorlevel 1 (
    echo ERROR: Scenario 2 failed!
    pause
    exit /b 1
)
echo [2/6] Complete!
echo.

echo [3/6] Running Scenario 3: Night Only (~15 seconds)
%LTSPICE% -b scenario_night_only.cir
if errorlevel 1 (
    echo ERROR: Scenario 3 failed!
    pause
    exit /b 1
)
echo [3/6] Complete!
echo.

echo [4/6] Running Scenario 4: 48hr Mixed (~60 seconds)
%LTSPICE% -b scenario_48hr_mixed.cir
if errorlevel 1 (
    echo ERROR: Scenario 4 failed!
    pause
    exit /b 1
)
echo [4/6] Complete!
echo.

echo [5/6] Running Scenario 5: Rainy Week (~5 minutes)
%LTSPICE% -b scenario_rainy_week.cir
if errorlevel 1 (
    echo ERROR: Scenario 5 failed!
    pause
    exit /b 1
)
echo [5/6] Complete!
echo.

echo [6/6] Running Scenario 6: Worst Case (~90 seconds)
%LTSPICE% -b scenario_worst_case.cir
if errorlevel 1 (
    echo ERROR: Scenario 6 failed!
    pause
    exit /b 1
)
echo [6/6] Complete!
echo.

echo ============================================================================
echo All simulations complete!
echo ============================================================================
echo.
echo Results saved in .log files:
echo   - scenario_sunny_24hr.log
echo   - scenario_cloudy_24hr.log
echo   - scenario_night_only.log
echo   - scenario_48hr_mixed.log
echo   - scenario_rainy_week.log
echo   - scenario_worst_case.log
echo.
echo To view results:
echo   1. Open each .log file in a text editor
echo   2. Scroll to bottom to see measurements
echo   3. Or open .raw files in LTspice to view waveforms
echo.
echo See LTSPICE_SIMULATION_SUITE_GUIDE.md for detailed analysis
echo ============================================================================
pause
