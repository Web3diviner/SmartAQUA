cmake_minimum_required(VERSION 3.10)
project(FishFeederSim)

set(CMAKE_CXX_STANDARD 11)

# Add SIMULATION define
add_definitions(-DSIMULATION)

# Compiler-specific flags
if(MSVC)
    # MSVC compiler flags
    set(CMAKE_CXX_FLAGS "${CMAKE_CXX_FLAGS} /W3")
else()
    # GCC/Clang compiler flags
    set(CMAKE_CXX_FLAGS "${CMAKE_CXX_FLAGS} -Wall -Wextra")
endif()

# Include directories
# Note: sim directory comes FIRST to prioritize stub headers over firmware headers
include_directories(
    ${PROJECT_SOURCE_DIR}/sim
    ${PROJECT_SOURCE_DIR}/include/hal
    ${PROJECT_SOURCE_DIR}/src/managers
)

# Source files
set(SOURCES
    sim/main_sim.cpp
    sim/hal_sim.cpp
    sim/DigitalTwin.cpp
    sim/NVSStorage_sim.cpp
    sim/SensorManager_sim.cpp
    src/managers/FeedingController.cpp
)

# Create executable
add_executable(fishfeeder_sim ${SOURCES})

# Link math library (not needed on Windows/MSVC)
if(NOT MSVC)
    target_link_libraries(fishfeeder_sim m)
endif()

# Print build info
message(STATUS "Building Smart Fish Feeder Simulation")
message(STATUS "C++ Standard: ${CMAKE_CXX_STANDARD}")
message(STATUS "Compiler: ${CMAKE_CXX_COMPILER}")
message(STATUS "Build type: ${CMAKE_BUILD_TYPE}")
