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

mkdir -p bin

CLASSPATH=$(find /app -name '*.jar' | tr '\n' ':')

if [ "$NEEDS_SHARED_LIBS" = "true" ]; then
    echo "Compiling with shared libraries..."
    javac -cp "$CLASSPATH" -d bin $(find src -name '*.java') $(find "$SHARED_LIBS_PATH" -name '*.java')
else
    echo "Compiling standalone module (no shared libraries needed)..."
    javac -cp "$CLASSPATH" -d bin $(find src -name '*.java')
fi

jar cfm "$(basename "$(pwd)").jar" "$MANIFEST_PATH" -C bin .

rm -rf bin
echo "Successfully built $(basename "$(pwd)").jar"
