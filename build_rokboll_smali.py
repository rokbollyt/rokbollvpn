import os
import subprocess
import shutil

android_jar = r"C:\Users\RokBolL\AppData\Local\Android\Sdk\platforms\android-34\android.jar"
d8_bat = r"C:\Users\RokBolL\AppData\Local\Android\Sdk\build-tools\36.0.0\d8.bat"
apktool_jar = r"D:\projects\go\apktool.jar"

base_dir = r"D:\projects\go\rokboll_java"
src_dir = os.path.join(base_dir, "src", "su", "happ", "proxyutility", "rokboll")
bin_dir = os.path.join(base_dir, "bin")
stubs_bin = os.path.join(base_dir, "stubs_bin")
dex_dir = os.path.join(base_dir, "dex")
apktool_out = os.path.join(base_dir, "apktool_out")
dest_smali = r"D:\projects\go\Happ_src\smali_classes2\su\happ\proxyutility\rokboll"

os.makedirs(bin_dir, exist_ok=True)
os.makedirs(dex_dir, exist_ok=True)

# 1. Compile Java with javac
print("Compiling Java files...")
java_files = [os.path.join(src_dir, f) for f in os.listdir(src_dir) if f.endswith(".java")]
cp = f"{android_jar};{stubs_bin};{bin_dir}"
cmd_javac = ["javac", "--release", "8", "-cp", cp, "-d", bin_dir] + java_files
res = subprocess.run(cmd_javac, capture_output=True, text=True)
if res.returncode != 0:
    print("javac error:")
    print(res.stderr)
    exit(1)
print("javac succeeded.")

# 2. Convert class files to dex with d8
print("Running d8...")
class_files = []
for root, dirs, files in os.walk(bin_dir):
    for f in files:
        if f.endswith(".class"):
            class_files.append(os.path.join(root, f))

cmd_d8 = [d8_bat, "--lib", android_jar, "--output", dex_dir] + class_files
res = subprocess.run(cmd_d8, capture_output=True, text=True)
if res.returncode != 0:
    print("d8 error:")
    print(res.stderr)
    exit(1)
print("d8 succeeded.")

# 3. Disassemble classes.dex with apktool
print("Converting dex to smali via apktool...")
import zipfile
zip_path = os.path.join(base_dir, "temp.zip")
apk_path = os.path.join(base_dir, "temp.apk")
with zipfile.ZipFile(zip_path, 'w') as z:
    z.write(os.path.join(dex_dir, "classes.dex"), "classes.dex")
if os.path.exists(apk_path):
    os.remove(apk_path)
os.rename(zip_path, apk_path)

cmd_apktool = ["java", "-jar", apktool_jar, "d", "-f", apk_path, "-o", apktool_out]
res = subprocess.run(cmd_apktool, capture_output=True, text=True)
if res.returncode != 0:
    print("apktool error:")
    print(res.stderr)
    exit(1)
print("apktool succeeded.")

# 4. Copy smali to Happ_src
src_smali = os.path.join(apktool_out, "smali", "su", "happ", "proxyutility", "rokboll")
for f in os.listdir(src_smali):
    shutil.copy2(os.path.join(src_smali, f), os.path.join(dest_smali, f))
    print("Copied", f)

print("Done recompiling Rokboll smali!")
