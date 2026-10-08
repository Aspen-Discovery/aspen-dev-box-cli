#!/bin/bash
set -e

SHARED_LIBS_PATH="${SHARED_LIBS_PATH:-/app/code/java_shared_libraries}"
MANIFEST_PATH=""

if [ -f "META-INF/MANIFEST.MF" ]; then
    MANIFEST_PATH="META-INF/MANIFEST.MF"
elif [ -f "src/META-INF/MANIFEST.MF" ]; then
    MANIFEST_PATH="src/META-INF/MANIFEST.MF"
else
    echo "ERROR: No MANIFEST.MF found in META-INF/ or src/META-INF/"
    exit 1
fi

echo "Using MANIFEST: $MANIFEST_PATH"

NEEDS_SHARED_LIBS=false
if grep -rq "import com.turning_leaf_technologies" src/ 2>/dev/null; then
    NEEDS_SHARED_LIBS=true
fi

copy_resources() {
    local source_root="$1"
    local target_dir="$2"
    if [ ! -d "$source_root" ]; then
        return 0
    fi
    (cd "$source_root" && find . -type f ! -name '*.java' -exec cp --parents -t "$target_dir" {} +)
}

mkdir -p bin
BIN_DIR="$(pwd)/bin"

MODULE_JAR="$(pwd)/$(basename "$(pwd)").jar"
CLASSPATH=$(find /app -name '*.jar' | grep -v "$MODULE_JAR" | tr '\n' ':')

SOURCE_ROOTS=(src)
if [ "$NEEDS_SHARED_LIBS" = "true" ]; then
    echo "Compiling with shared libraries..."
    SOURCE_ROOTS+=("$SHARED_LIBS_PATH/src")
else
    echo "Compiling standalone module (no shared libraries needed)..."
fi

javac -g --release 11 -cp "$CLASSPATH" -d bin $(find "${SOURCE_ROOTS[@]}" -name '*.java')

for source_root in "${SOURCE_ROOTS[@]}"; do
    copy_resources "$source_root" "$BIN_DIR"
done

jar cfm "$(basename "$(pwd)").jar" "$MANIFEST_PATH" -C bin .

rm -rf bin
echo "Successfully built $(basename "$(pwd)").jar"
