# Guía de Configuración: Engram Cloud (Colaborativo)

Esta guía explica cómo configurar y desplegar el ecosistema de Engram en modo colaborativo (Nube), diferenciando entre la gestión del servidor y la configuración de los equipos cliente.

---

## 🛠️ 1. Configuración del Administrador (Server Side)

El administrador es responsable de mantener la infraestructura y gestionar los accesos a la base de datos.

### Requisitos Previos
- Docker y Docker Compose instalados.
- Puerto `18080` (o el configurado) abierto para tráfico entrante.

### Despliegue de la Infraestructura
1. **Configurar el archivo `.env`**:
   Copia el archivo `.env.example` (si existe) o crea uno con estos valores mínimos:
   ```env
   # Configuración de Postgres
   POSTGRES_USER=engram
   POSTGRES_PASSWORD=tu_password_seguro
   POSTGRES_DB=engram_cloud
   POSTGRES_IMAGE=postgres:16-alpine

   # Configuración del Servidor Engram
   ENGRAM_PORT=18080
   ENGRAM_CLOUD_TOKEN=token_de_administrador_para_bootstrap
   ENGRAM_JWT_SECRET=un_secreto_muy_largo_y_aleatorio
   ENGRAM_CLOUD_ALLOWED_PROJECTS=proyecto1,proyecto2,engram
   ```

2. **Levantar los servicios**:
   ```bash
   docker compose -f docker-compose.cloud.yml up -d --build
   ```

### Gestión de Usuarios y Proyectos (Bootstrap)
Actualmente, la gestión se realiza directamente sobre la base de datos (vía SQL). Para dar de alta a un nuevo miembro del equipo:

1. **Crear el usuario**:
   ```sql
   INSERT INTO cloud_users (username, email, api_key) 
   VALUES ('nombre_usuario', 'email@empresa.com', 'api_key_unica_para_el_usuario');
   ```

2. **Crear el proyecto**:
   ```sql
   INSERT INTO cloud_projects (id, name, owner_id) 
   VALUES ('id_del_proyecto', 'Nombre del Proyecto', id_usuario_owner);
   ```

3. **Asignar miembros al proyecto**:
   ```sql
   INSERT INTO cloud_project_members (project_id, user_id, role) 
   VALUES ('id_del_proyecto', id_usuario, 'member');
   ```

---

## 💻 2. Configuración de Equipos Clientes (Team Side)

Cada desarrollador en el equipo debe configurar su entorno local para conectarse al servidor central.

### Paso 1: Obtener el binario
El administrador debe proveer el binario compilado o el equipo debe compilarlo desde el source:
```powershell
go build -o engram.exe ./cmd/engram
```

### Paso 2: Configurar la identidad
Cada desarrollador debe crear un archivo de configuración en su carpeta personal. 

**Ruta en Windows:** `%USERPROFILE%\.engram\config.json`  
**Ruta en Linux/Mac:** `~/.engram/config.json`

Contenido del archivo:
```json
{
  "base_url": "http://tu-servidor-engram:18080",
  "api_key": "la_api_key_que_te_paso_el_admin"
}
```

### Paso 3: Validar la conexión
Ejecutar el comando de estadísticas para confirmar que hay comunicación con la nube:
```bash
./engram stats
```

---

## 🚀 3. Uso Colaborativo

Una vez configurados, todos los miembros con acceso al mismo `project_id` compartirán la memoria en tiempo real.

- **Guardar una memoria compartida**:
  ```bash
  ./engram save "Título" "Contenido importante" --type "decision"
  ```
- **Buscar memorias del equipo**:
  ```bash
  ./engram search "termino de busqueda"
  ```
- **Explorar la línea de tiempo**:
  Usa el TUI para ver qué estuvieron haciendo tus compañeros:
  ```bash
  ./engram tui
  ```

---

## ⚠️ Consideraciones de Seguridad
1. **API Keys**: Son personales e intransferibles. Si una key se filtra, el administrador debe invalidarla en la tabla `cloud_users`.
2. **HTTPS**: En entornos de producción, se recomienda poner un proxy inverso (Nginx/Caddy) delante del contenedor de Engram para habilitar SSL/TLS.
3. **Membresías**: Un usuario no podrá ver ni guardar memorias en un proyecto a menos que haya sido explícitamente añadido a la tabla `cloud_project_members`.
