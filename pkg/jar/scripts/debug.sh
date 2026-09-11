#!/bin/bash
set -e

if pgrep -f "jdwp=transport=dt_socket,server=y,suspend=y,address=\*:${DEBUG_PORT}" > /dev/null; then
    echo "A debug session is already listening on port ${DEBUG_PORT}; stop it first (adb shell, then: pkill -f jdwp)"
    exit 1
fi

cd "/usr/local/aspen-discovery/code/${MODULE}"
if [ ! -d src ]; then
    echo "ERROR: no src directory in $(pwd)"
    exit 1
fi

BIN_DIR="/tmp/aspen-debug/${MODULE}"
rm -rf "${BIN_DIR}"
mkdir -p "${BIN_DIR}"

CLASSPATH="${BIN_DIR}:$(find /usr/local/aspen-discovery/code /usr/local/aspen-discovery/sites/default -name '*.jar' | tr '\n' ':')"
SOURCES=$(find src -name '*.java')
if [ "${NEEDS_SHARED_LIBS}" = "true" ]; then
    SOURCES="${SOURCES} $(find /usr/local/aspen-discovery/code/java_shared_libraries -name '*.java')"
fi

echo "Compiling ${MODULE} for debugging..."
javac -cp "${CLASSPATH}" -d "${BIN_DIR}" ${SOURCES}

echo "Waiting for a debugger to attach on port ${DEBUG_PORT} (${MAIN_CLASS})..."
exec java -agentlib:jdwp=transport=dt_socket,server=y,suspend=y,address=*:${DEBUG_PORT} \
    -cp "${CLASSPATH}" "${MAIN_CLASS}" "${SITE_NAME}" "$@"
