# ZeroCodex - AI Coding Agent

ZeroCodex es una aplicación de escritorio que utiliza inteligencia artificial para ayudar a los desarrolladores a modificar y mantener proyectos de código. Actúa como un asistente de programación que puede leer, analizar y modificar archivos de código en respuesta a solicitudes en lenguaje natural.

## 🚀 Características Principales

### 🤖 Asistente de IA Integrado

- **DeepSeek AI**: Integración con el modelo DeepSeek para análisis y generación de código
- **Contexto inteligente**: Analiza automáticamente la estructura del proyecto
- **Detección de tipo de proyecto**: Identifica Go, Flutter/Dart, Node.js, Python, Rust y más

### 📁 Gestión de Proyectos

- **Selección de proyectos**: Interfaz gráfica para seleccionar carpetas de proyectos
- **Persistencia**: Almacena proyectos recientes en base de datos SQLite
- **Validación automática**: Detecta si una carpeta contiene un proyecto válido

### 🔧 Herramientas de Desarrollo

- **Lectura de archivos**: Puede leer archivos específicos o rangos de líneas
- **Escritura de archivos**: Modifica archivos existentes o crea nuevos
- **Análisis de cambios**: Muestra diferencias de Git después de las modificaciones
- **Sugerencias inteligentes**: Sugiere archivos relevantes basados en la solicitud

### 🎨 Interfaz de Usuario

- **Aplicación de escritorio**: Construida con Wails
- **Frontend moderno**: Interfaz en Svelte con HTML, CSS y JavaScript
- **Chat interactivo**: Interfaz de conversación estilo chat
- **Panel lateral**: Lista de proyectos recientes
- **Estado en tiempo real**: Muestra el progreso de las operaciones

## 🏗️ Arquitectura

ZeroCodex sigue una arquitectura limpia con separación clara de responsabilidades:

### Capas de la Aplicación

```
┌─────────────────────────────────────────┐
│           Interfaz de Usuario           │
│     (Wails + frontend Svelte/Vite)      │
└─────────────────────────────────────────┘
                   │
┌─────────────────────────────────────────┐
│          Lógica de Aplicación           │
│       (internal/application/)           │
│  • Chat - Procesamiento de solicitudes  │
│  • Project Intelligence - Análisis      │
│  • Select Project - Gestión proyectos   │
└─────────────────────────────────────────┘
                   │
┌─────────────────────────────────────────┐
│           Dominio / Puertos             │
│         (internal/domain/)              │
│  • ProjectRepository - Operaciones FS   │
│  • LLMClient - Interfaz IA              │
│  • ProjectStore - Almacenamiento        │
└─────────────────────────────────────────┘
                   │
┌─────────────────────────────────────────┐
│          Adaptadores Concretos          │
│       (internal/adapters/)              │
│  • filesystem/ - Operaciones archivos   │
│  • llm/ - Cliente DeepSeek              │
│  • storage/ - SQLite para proyectos     │
└─────────────────────────────────────────┘
```

### Flujo de Trabajo

1. **Selección de proyecto**: El usuario selecciona una carpeta de proyecto
2. **Análisis**: ZeroCodex analiza la estructura y detecta el tipo de proyecto
3. **Solicitud**: El usuario escribe una solicitud en lenguaje natural
4. **Contexto**: Se construye un contexto inteligente con archivos relevantes
5. **Procesamiento**: La IA analiza y genera modificaciones
6. **Ejecución**: Se aplican los cambios al sistema de archivos
7. **Feedback**: Se muestran los resultados y cambios realizados

## 📦 Instalación y Uso

### Requisitos Previos

- **Go 1.25.6** o superior
- **Clave API de DeepSeek** (se configura dentro de la app)

### Configuración

1. **Clonar el repositorio**:

   ```bash
   git clone https://github.com/cobyzero/zerocodex.git
   cd zerocodex
   ```

2. **Instalar dependencias del frontend**:
   ```bash
   cd frontend
   npm install
   cd ..
   ```

3. **Ejecutar en desarrollo**:
   ```bash
   wails dev
   ```

4. **Construir binario**:
   ```bash
   wails build
   ```

### Uso Básico

1. **Iniciar la aplicación**: Ejecutar `wails dev` o el binario generado
2. **Configurar la API key**: Guardarla desde la pantalla inicial
3. **Seleccionar proyecto**: Abrir la carpeta con el selector nativo de escritorio
4. **Escribir solicitud**: En el campo de texto, describir lo que quieres modificar
5. **Enviar**: Presionar el botón "Enviar" o `Ctrl+Enter`
6. **Revisar cambios**: Los cambios se aplican automáticamente y se muestran en el chat

### Ejemplos de Solicitudes

- "Agrega una función que valide emails en el archivo utils.go"
- "Crea un nuevo componente de botón en Flutter con estilo material"
- "Corrige el error de sintaxis en main.py línea 45"
- "Refactoriza la función de login para usar async/await"
- "Agrega tests para el módulo de autenticación"

## 🔧 Configuración Avanzada

### Variables de Entorno

No son necesarias para el flujo principal. La API key se guarda en el llavero del sistema desde la interfaz.

### Almacenamiento

- **Base de datos**: SQLite en `~/.config/zerocodex/projects.db`
- **Configuración**: Archivos de configuración en el directorio de usuario
- **Logs**: Logs de aplicación en stdout

## 🛠️ Desarrollo

### Estructura del Proyecto

```
zerocodex/
├── main.go                 # Punto de entrada Wails
├── app.go                  # Métodos Go expuestos al frontend
├── frontend/               # UI Svelte/Vite
├── wails.json              # Configuración Wails
├── internal/
│   ├── adapters/           # Adaptadores concretos
│   │   ├── filesystem/     # Operaciones de sistema de archivos
│   │   ├── llm/           # Cliente DeepSeek
│   │   ├── storage/       # Almacenamiento SQLite
│   │   └── ui/            # UI Fyne anterior, ya no usada por defecto
│   ├── application/        # Lógica de aplicación
│   │   ├── chat.go        # Procesamiento de chat
│   │   ├── project_intelligence.go # Análisis de proyectos
│   │   └── select_project.go # Gestión de proyectos
│   └── domain/            # Dominio y puertos
│       ├── ports.go       # Interfaces principales
│       ├── project.go     # Entidad proyecto
│       └── project_store.go # Almacenamiento de proyectos
├── go.mod                 # Dependencias Go
└── README.md             # Documentación
```

### Dependencias Principales

- **Wails**: Shell de escritorio y bindings Go <-> frontend
- **Svelte + Vite**: Interfaz moderna y mantenible
- **SQLite**: Base de datos embebida para persistencia

### Construir desde Fuente

```bash
# Desarrollo
wails dev

# Build de escritorio
wails build
```

## 🤝 Contribuir

1. **Fork** el repositorio
2. **Crear rama** para tu feature (`git checkout -b feature/AmazingFeature`)
3. **Commit** tus cambios (`git commit -m 'Add some AmazingFeature'`)
4. **Push** a la rama (`git push origin feature/AmazingFeature`)
5. **Abrir Pull Request**

### Convenciones de Código

- **Go**: Seguir las convenciones estándar de Go
- **Arquitectura**: Respetar la separación de capas (dominio, aplicación, adaptadores)
- **Tests**: Agregar tests para nuevas funcionalidades
- **Documentación**: Actualizar documentación relevante

## 📄 Licencia

Este proyecto está bajo la Licencia MIT. Ver el archivo LICENSE para más detalles.

## 🙏 Agradecimientos

- **DeepSeek AI** por proporcionar la API de inteligencia artificial
- **Comunidad Go** por las herramientas y bibliotecas de calidad

## 🖼️ Imágenes de la Aplicación

Aquí puedes ver capturas de pantalla de ZeroCodex en acción:

![Interfaz principal de ZeroCodex](https://raw.githubusercontent.com/cobyzero/zerocodex/refs/heads/main/1.png)

## 🐛 Reportar Problemas

Si encuentras algún problema o tienes sugerencias:

1. **Buscar** en issues existentes
2. **Crear nuevo issue** con:
   - Descripción clara del problema
   - Pasos para reproducir
   - Comportamiento esperado vs actual
   - Capturas de pantalla (si aplica)

---

**ZeroCodex** - Tu asistente de programación inteligente para modificar proyectos de código con IA.
