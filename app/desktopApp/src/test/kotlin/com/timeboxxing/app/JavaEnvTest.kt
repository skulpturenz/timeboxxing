package com.timeboxxing.app

import kotlin.test.Test
import kotlin.test.assertEquals

class JavaEnvTest {
    @Test
    fun parsesSupportedValuesCaseInsensitively() {
        assertEquals(JavaEnv.Production, JavaEnv.parse("production"))
        assertEquals(JavaEnv.Development, JavaEnv.parse("Development"))
        assertEquals(JavaEnv.Test, JavaEnv.parse(" TEST "))
        assertEquals(JavaEnv.Local, JavaEnv.parse("local"))
    }

    @Test
    fun rejectsUnsupportedValues() {
        assertEquals(null, JavaEnv.parse("staging"))
        assertEquals(null, JavaEnv.parse(""))
    }
}
