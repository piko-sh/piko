import org.jetbrains.grammarkit.tasks.GenerateLexerTask

plugins {
    id("java")
    id("org.jetbrains.kotlin.jvm") version "2.3.0"
    id("org.jetbrains.intellij.platform") version "2.11.0"
    id("org.jetbrains.grammarkit") version "2023.3.0.1"
    id("io.gitlab.arturbosch.detekt") version "1.23.8"
    id("org.jetbrains.kotlinx.kover") version "0.9.1"
}

group = "io.politepixels"
version = "0.1.0"

repositories {
    mavenCentral()
    intellijPlatform {
        defaultRepositories()
    }
}

dependencies {
    testImplementation("junit:junit:4.13.2")

    intellijPlatform {
        goland("2025.3")
        testFramework(org.jetbrains.intellij.platform.gradle.TestFrameworkType.Platform)
        bundledPlugin("org.jetbrains.plugins.go")
        bundledPlugin("com.intellij.css")
        bundledPlugin("JavaScript")
        plugin("com.redhat.devtools.lsp4ij:0.19.1")
    }
}

intellijPlatform {
    pluginConfiguration {
        ideaVersion {
            sinceBuild = "253"
        }
        changeNotes = """
            Initial release with:
            - Go language injection in script blocks
            - CSS language injection in style blocks
            - TypeScript injection in JavaScript script blocks
            - LSP support for template blocks
        """.trimIndent()
    }

    publishing {
        token = providers.environmentVariable("PUBLISH_TOKEN")
    }
}

grammarKit {
    jflexRelease.set("1.9.1")
    grammarKitRelease.set("2021.1.2")
}

tasks.register<GenerateLexerTask>("generatePKLexer") {
    sourceFile.set(file("src/main/grammars/PKLexer.flex"))
    targetOutputDir.set(file("src/main/java/io/politepixels/gen/pk"))
    purgeOldFiles.set(true)
}

sourceSets {
    main {
        java {
            srcDirs("src/main/java/io/politepixels/gen/pk")
        }
    }
}

val copyTypeDefinitions by tasks.registering(Copy::class) {
    from("${rootProject.projectDir}/../../internal/typegen/typegen_frontend/built/piko-ide.d.ts")
    from("${rootProject.projectDir}/../../internal/typegen/typegen_adapters/embedded/piko-actions-stub.d.ts") {
        rename("piko-actions-stub.d.ts", "piko-actions.d.ts")
    }
    into(layout.projectDirectory.dir("src/main/resources/types"))
}

tasks.named("processResources") {
    dependsOn(copyTypeDefinitions)
}

tasks.named("compileJava") {
    dependsOn("generatePKLexer")
}

tasks.named("compileKotlin") {
    dependsOn("generatePKLexer")
}

tasks {
    withType<JavaCompile> {
        sourceCompatibility = "21"
        targetCompatibility = "21"
    }
}

kotlin {
    compilerOptions {
        jvmTarget.set(org.jetbrains.kotlin.gradle.dsl.JvmTarget.JVM_21)
    }
}

detekt {
    buildUponDefaultConfig = true
    config.setFrom(files("$projectDir/detekt.yml"))
    source.setFrom(files("src/main/kotlin"))
}

tasks.named("detekt") {
    dependsOn("generatePKLexer")
}
