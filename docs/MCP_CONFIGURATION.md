# Guía de Configuración: Servidor MCP para Agentes de IA

Esta guía explica cómo conectar tus agentes de IA (Claude, GPT, Cursor, etc.) a tu servidor de Engram Cloud usando el protocolo MCP (Model Context Protocol).

---

## 🚀 Concepto Clave
El binario de `engram` tiene un servidor MCP integrado. Cuando un agente de IA lo arranca, el binario lee tu archivo `~/.engram/config.json` y se conecta automáticamente a tu nube. 

**No importa qué agente uses, todos compartirán la misma base de conocimiento centralizada.**

---

## 👤 1. Configuración en Claude Desktop

Para que Claude (la app de escritorio) pueda usar Engram Cloud:

1. Abre el archivo de configuración de Claude:
   - **Windows:** `%APPDATA%\Claude\claude_desktop_config.json`
   - **macOS:** `~/Library/Application Support/Claude/claude_desktop_config.json`

2. Agrega `engram` a la sección de `mcpServers`:
   ```json
   {
     "mcpServers": {
       "engram": {
         "command": "C:\\Ruta\\A\\Tu\\engram.exe",
         "args": ["mcp-server"]
       }
     }
   }
   ```
   *(Asegúrate de poner la ruta absoluta correcta a tu binario `engram.exe`)*.

3. Reinicia Claude Desktop. Verás un icono de "martillo" o "herramientas" que indica que Engram está activo.

---

## 🖱️ 2. Configuración en Cursor / VS Code

Si usas **Cursor** o **VS Code** con la extensión de MCP:

1. Ve a los Ajustes de Cursor (`Ctrl + Shift + J`) -> **General** -> **MCP**.
2. Haz clic en **+ Add New MCP Server**.
3. Configura los campos:
   - **Name:** `Engram`
   - **Type:** `command`
   - **Command:** `C:\Ruta\A\Tu\engram.exe mcp-server`
4. Haz clic en **Save**. Verás que Cursor escanea las herramientas disponibles (`mem_save`, `mem_search`, etc.).

---

## 🛠️ 3. Configuración para otros Agentes (CLI / Custom)

Si estás usando un agente basado en línea de comandos o tu propio desarrollo, simplemente invoca el binario:

```bash
engram mcp-server
```

El servidor utiliza el estándar **JSON-RPC sobre Stdio**, que es compatible con cualquier cliente MCP moderno.

---

## 💡 Tips de Uso Colaborativo con la IA

Cuando hables con tu agente, podés decirle cosas como:

- *"Recordá que para este proyecto usamos la arquitectura hexagonal"* -> El agente usará `mem_save` y lo guardará en la **Nube**.
- *"¿Qué decisiones tomamos ayer sobre la base de datos?"* -> El agente usará `mem_search`, consultará la **Nube** y te responderá aunque lo hayas guardado desde otra máquina o lo haya guardado otro compañero.

### Variables de Entorno (Avanzado)
Si querés que un agente específico use una configuración distinta, podés pasarle variables de entorno en la configuración del servidor MCP:

```json
"env": {
  "ENGRAM_API_URL": "http://otra-nube.com",
  "ENGRAM_API_KEY": "otro-token"
}
```

---

¡Listo! Con esto, tus agentes de IA ahora tienen memoria compartida a largo plazo. 🥂🚀
