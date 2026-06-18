package com.timeboxxing.app

interface Platform {
    val name: String
}

expect fun getPlatform(): Platform