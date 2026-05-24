cmake_minimum_required(VERSION 3.5)
project(ru.pmifi.Aseek CXX)

find_package (Qt5 COMPONENTS Core Network Qml Gui Quick LinguistTools REQUIRED)

include(FindPkgConfig)
pkg_search_module(AURORA auroraapp REQUIRED)
pkg_search_module(AURORA auroraapp_i18n REQUIRED)

set(CMAKE_AUTOMOC ON)
set(CMAKE_INCLUDE_CURRENT_DIR ON)


set(short_name Aseek)

file(GLOB TsFiles "translations/*.ts")
qt5_add_translation(QmFiles ${TsFiles})

set(SOURCES
    qml/${short_name}.qml
    qml/cover/DefaultCoverPage.qml
    qml/pages/AboutPage.qml
    qml/pages/MainPage.qml
    src/main.cpp
)

set(CMAKE_SKIP_RPATH FALSE)
set(CMAKE_BUILD_WITH_INSTALL_RPATH TRUE)
set(CMAKE_INSTALL_RPATH "${CMAKE_INSTALL_PREFIX}/share/${PROJECT_NAME}/lib")


set(LLAMA_BUILD_COMMON      ON CACHE BOOL "" FORCE)
set(LLAMA_BUILD_TOOLS       ON CACHE BOOL "" FORCE)
set(LLAMA_BUILD_EXAMPLES    OFF CACHE BOOL "" FORCE)
set(LLAMA_TOOLS_INSTALL     ON CACHE BOOL "" FORCE)
set(LLAMA_BUILD_TESTS       OFF CACHE BOOL "" FORCE)
SET(LLAMA_BUILD_SERVER      ON CACHE BOOL "" FORCE)

set(BUILD_SHARED_LIBS       OFF CACHE BOOL "" FORCE)
set(GGML_NATIVE             OFF CACHE BOOL "" FORCE)
set(GGML_OPENMP             OFF  CACHE BOOL "" FORCE)
set(GGML_BLAS               OFF CACHE BOOL "" FORCE)

set(GGML_CUDA               OFF CACHE BOOL "" FORCE)
set(GGML_METAL              OFF CACHE BOOL "" FORCE)
set(GGML_VULKAN             OFF CACHE BOOL "" FORCE)
set(GGML_OPENCL             OFF CACHE BOOL "" FORCE)
set(GGML_HIP                OFF CACHE BOOL "" FORCE)
set(GGML_SYCL               OFF CACHE BOOL "" FORCE)

set(GGML_CPU_ARM_NEON    OFF CACHE BOOL "" FORCE)
set(LLAMA_LLAMAFILE OFF CACHE BOOL "" FORCE)
set(GGML_NATIVE          OFF CACHE BOOL "" FORCE)
set(GGML_CPU_ARM_NEON    OFF CACHE BOOL "" FORCE)
set(GGML_ARM7            ON  CACHE BOOL "" FORCE)
set(GGML_CPU_ALL_VARIANTS OFF CACHE BOOL "" FORCE)

set(GGML_LLAMAFILE OFF CACHE BOOL "" FORCE)



# set(CMAKE_C_FLAGS   "${CMAKE_C_FLAGS} -march=armv8-a -mtune=cortex-a53 -O3 -ffast-math -ftree-vectorize -fomit-frame-pointer" CACHE STRING "" FORCE)
# set(CMAKE_CXX_FLAGS "${CMAKE_CXX_FLAGS} -march=armv8-a -mtune=cortex-a53 -O3 -ffast-math -ftree-vectorize -fomit-frame-pointer" CACHE STRING "" FORCE)

add_subdirectory(llama.cpp/)


add_executable(${PROJECT_NAME} ${SOURCES} ${QmFiles})
target_compile_definitions(${PROJECT_NAME} PRIVATE
    $<$<OR:$<CONFIG:Debug>,$<CONFIG:RelWithDebInfo>>:QT_QML_DEBUG>
)
target_include_directories(${PROJECT_NAME} PRIVATE
    $<BUILD_INTERFACE:
    ${AURORA_INCLUDE_DIRS}
>)
target_link_libraries(${PROJECT_NAME} PRIVATE
    Qt5::Quick
    ${AURORA_LDFLAGS}
)

install(TARGETS ${PROJECT_NAME}
    RUNTIME DESTINATION ${CMAKE_INSTALL_BINDIR}
)
install(DIRECTORY qml
    DESTINATION share/${PROJECT_NAME}
)

foreach(_file IN LISTS QmFiles)
    get_filename_component(_filename "${_file}" NAME)
    string(REPLACE "${short_name}" "${PROJECT_NAME}" _full_filename "${_filename}")
    install(FILES ${_file} DESTINATION share/${PROJECT_NAME}/translations RENAME ${_full_filename})
endforeach()

set(IconSize 86x86 108x108 128x128 172x172)
foreach(_size IN LISTS IconSize)
    install(FILES icons/${_size}/${PROJECT_NAME}.png DESTINATION share/icons/hicolor/${_size}/apps)
endforeach()

install(FILES ${PROJECT_NAME}.desktop
    DESTINATION share/applications
)

# Get the other files reachable from the project tree in Qt Creator
file(GLOB_RECURSE IconFiles "icons" "*.png")
set(RESOURCE_FILES ${IconFiles})

add_custom_target(distfiles
    SOURCES
        ${PROJECT_NAME}.desktop
        rpm/${PROJECT_NAME}.spec
        rpm/${PROJECT_NAME}.changes.in
        rpm/${PROJECT_NAME}.changes.run.in
        ${TsFiles}
        ${IconFiles})

file(WRITE "${CMAKE_BINARY_DIR}/QtCreatorDeployment.txt"
    "${CMAKE_INSTALL_PREFIX}
${CMAKE_BINARY_DIR}/${PROJECT_NAME}:bin
")
