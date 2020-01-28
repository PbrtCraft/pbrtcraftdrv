# pbrtcraftdrv

Pbrtcraftdrv Provide Operate GUI for PbrtCraft.

![](demo.png)

## Build

```bash
$ cd build
$ ./build ../../pbrtcraftdrv-build
```

## Pages 

* Dashboard: Main function: using mc2pbrt and pbrt
* Result: Show last rendering result
* Files: Show `workdir` file tree
* Logs: Show logging files

## Config

### Config File

* Example Config File: [configs/appconfig.yaml](configs/appconfig.yaml)

#### dictionary content

- mcw_driver:
  - workdir: Working diretcory.
  - mc2pbrt_main: mc2pbrt execute file. It can ba an exe file or 
  - pbrt_bin: Compiled pbrt-v3-minecraft binary.
  - log_dir: Directory for log file.
- python_file:
  - camera: Path tp mc2pbrt camera's file.
  - phenomenon: Path tp mc2pbrt phenomenon's file.
  - method: Path tp mc2pbrt method's file.
- minecraft:
  - directory: Path to minecraft world directory. Leave empty for auto detection.
- srv:
  - port: Server Port.
