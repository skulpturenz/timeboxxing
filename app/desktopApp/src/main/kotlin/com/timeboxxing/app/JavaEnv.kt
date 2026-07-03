package com.timeboxxing.app

internal enum class JavaEnv(val value: String) {
    Production("production"),
    Development("development"),
    Test("test"),
    Local("local"),
    ;

    companion object {
        fun parse(value: String?): JavaEnv? {
            val normalized = value
                ?.trim()
                ?.lowercase()
                ?: return null
            return entries.firstOrNull { it.value == normalized }
        }
    }
}

internal const val JavaEnvVar = "JAVA_ENV"
internal const val GoEnvVar = "GO_ENV"
