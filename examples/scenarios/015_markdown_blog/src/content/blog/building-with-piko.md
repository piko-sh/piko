---
title: Building with Piko
description: A look at what makes Piko different from other web frameworks, and why server-side rendering deserves a second chance.
date: 2026-02-05
tags:
  - piko
  - server-side rendering
---

# Building with Piko

Piko is a server-side rendering framework written in Go. It takes a different approach to modern web development:

- **Single-file components** - templates, styles, and logic live together
- **File-based routing** - your directory structure is your URL structure
- **Go on the server** - type-safe rendering with compiled performance
- **Web Components on the client** - lightweight virtual DOM for Piko Components, no framework runtime

## Why Server-Side Rendering?

The pendulum has swung back. After years of shipping megabytes of JavaScript to render a paragraph of text, developers are rediscovering the value of sending HTML from the server.

> The fastest page load is the one where the browser receives ready-to-render HTML.

Server-side rendering is not a step backward. It is the foundation that client-side interactivity should be built on top of, not instead of.

## Getting Started

A Piko project starts with a `pages/` directory and a `main.go` file. Write your first `.pk` template, run the generator, and you have a working site. No configuration files, no build tool chains, no decision fatigue.

The framework handles the rest: routing, asset processing, CSS scoping, and deployment packaging.
