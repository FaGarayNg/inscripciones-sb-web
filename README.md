# Inscripciones Simón Bolívar 4-084

Sistema web para los ingresantes al ciclo lectivo escolar de la escuela secundaria Simón Bolívar 4-084.

## Descripción

En este sistema los ingresantes podrán:

- **Confirmar sus bancos**: Verificar su inscripción y reservar su lugar
- **Formar grupos**: Crear o unirse a grupos para comenzar su cursado

## Estructura del Proyecto

```
inscripciones-sb-web/
├── frontend/          # Aplicación Angular
│   ├── src/
│   │   ├── app/
│   │   │   ├── features/    # Componentes por funcionalidad
│   │   │   ├── model/       # Modelos de datos
│   │   │   ├── service/     # Servicios HTTP
│   │   │   └── shared/      # Componentes compartidos
│   └── ...
└── backend/           # API Go + MongoDB
    ├── config/        # Configuración de base de datos
    ├── models/        # Modelos de datos
    ├── repository/    # Operaciones CRUD
    ├── handlers/      # Manejadores HTTP
    └── router/        # Rutas de la API
```

## Tecnologías

| Capa | Tecnología |
|------|------------|
| Frontend | Angular 22 |
| Backend | Go 1.26+ |
| Base de datos | MongoDB Atlas |

## Configuración

### Frontend

```bash
cd frontend
npm install
ng serve
```

### Backend

```bash
cd backend
export MONGODB_URI="mongodb+srv://usuario:password@cluster0.xxxxx.mongodb.net/"
go run main.go
```

## Endpoints de la API

### Públicas (sin autenticación)

| Método | Ruta | Descripción |
|--------|------|-------------|
| POST | `/api/login` | Iniciar sesión del administrador |
| GET | `/api/students` | Listar estudiantes |
| GET | `/api/students/{id}` | Obtener estudiante |
| POST | `/api/students/{id}/confirm` | Confirmar banco (flujo del ingresante) |
| GET | `/api/groups` | Listar grupos (con nombres, sin correos) |
| GET | `/api/groups/{id}` | Obtener grupo |
| POST | `/api/groups/identify` | Identificarse con DNI + email (devuelve token de alumno) |
| GET | `/api/settings` | Consultar si la formación de grupos está habilitada |
| GET | `/api/abanderados` | Listar abanderados y escoltas titulares |

Los listados públicos de alumnos y grupos **no incluyen correos electrónicos**: el email
solo se expone en los endpoints `/api/admin` y se guarda cuando el ingresante confirma su banco.

### Protegidas (requieren `Authorization: Bearer <token>`)

| Método | Ruta | Descripción |
|--------|------|-------------|
| GET | `/api/admin/students` | Listar estudiantes (dashboard) |
| POST | `/api/admin/students` | Crear estudiante |
| PUT | `/api/admin/students/{id}` | Actualizar estudiante |
| DELETE | `/api/admin/students/{id}` | Eliminar estudiante |
| POST | `/api/admin/students/import` | Importar alumnos desde Excel |
| GET | `/api/admin/students/export` | Descargar listado ordenado en `.xlsx` (Nombre, Email, Instancia, Titulo, Grupo) |
| POST | `/api/admin/abanderados/import` | Importar la lista de abanderados desde Excel |
| DELETE | `/api/admin/abanderados/{id}` | Quitar un abanderado de la lista |
| GET | `/api/admin/groups` | Listar grupos (dashboard) |
| DELETE | `/api/admin/groups/{id}` | Eliminar grupo (cancela invitaciones y libera a sus integrantes) |
| DELETE | `/api/admin/groups/{id}/members/{memberId}` | Quitar integrante |

> Toda escritura del panel requiere token de administrador. Solo la lectura de listados,
> la identificación/creación de grupos y la confirmación de banco son públicas.
> La confirmación pública pide el DNI además del ID del alumno y no se puede repetir.

### Flujo de grupos (token de alumno)

No hay login para ingresantes: en `/grupos` se identifican con su **DNI + el correo con el que
confirmaron el banco**. El backend devuelve un JWT de alumno que se envía en la cabecera
`X-Student-Token` (distinto del `Bearer` de administrador).

| Método | Ruta | Descripción |
|--------|------|-------------|
| GET | `/api/groups/me` | Mi grupo, invitaciones recibidas y enviadas |
| POST | `/api/groups` | Crear grupo: `{solo: boolean, invitees: string[]}` (IDs de alumno) |
| POST | `/api/groups/invitations/{id}/accept` | Aceptar invitación |
| POST | `/api/groups/invitations/{id}/reject` | Rechazar invitación |
| POST | `/api/groups/invitations/{id}/cancel` | Cancelar invitación pendiente (solo el líder) |

Reglas: máximo **3 integrantes contando al líder**; se puede invitar a registrados confirmados
sin grupo (hasta 2); en la web se busca al invitado **por nombre** y se envía su ID (así no hay
ambigüedad entre alumnos con el mismo nombre); al aceptar se cancelan las demás invitaciones
pendientes; entrar **en solitario es definitivo**. Todo depende de `groupsEnabled`: si está en `false`, `/grupos` solo
muestra el aviso de "Próximamente", `identify` devuelve 403 y el frontend revalida el flag
antes de cada acción (así reacciona aunque el admin lo cambie con la página abierta).
No se envían correos: las invitaciones se ven dentro de `/grupos`.

### Plantillas de Excel

Las plantillas se descargan desde el panel Admin (sección Alumnos) y se sirven
como archivos estáticos desde `frontend/public/`:

- `plantilla-alumnos.xlsx`: columnas DNI, Nombre Completo, Instancia, Titulo y
  Escuela. Las columnas C, D y E tienen listas desplegables. Se regenera con
  `cd backend && go run ./cmd/gen-template`.
- `plantilla-abanderados.xlsx`: columnas Nombre y apellido, DNI y Curso (las tres
  obligatorias; Curso es texto libre, p. ej. `1° A`). Se genera con
  `cd backend && go run ./cmd/gen-abanderados-template`.

En ambas, la fila 1 es el encabezado y se ignora; los datos empiezan en la fila 2.
Los DNI repetidos se omiten y se informan en el resultado de la importación.
Las instancias se normalizan al importar o guardar y se guardan como `Primera`,
`Segunda`, `Tercera` o `Cuarta` (también se aceptan las formas largas, p. ej.
`Primera instancia`).

La exportación del panel genera `listado-alumnos.xlsx` con encabezados estilizados y
filtro. El orden es: ingresantes con título CUD/Discapacidad (filas amarillas y en
negrita), luego cada grupo por fecha de creación con su líder e integrantes, y por
 último los alumnos sin grupo en orden alfabético. La columna Grupo indica
`En solitario`, `Líder · Grupo de X` o `Integrante · Grupo de X`.

## Variables de entorno

### Backend (`backend/.env`, ver `backend/.env.example`)

| Variable | Descripción |
|----------|-------------|
| `MONGODB_URI` | URI de MongoDB Atlas |
| `ADMIN_USERNAME` | Usuario único del dashboard |
| `ADMIN_PASSWORD` | Hash **bcrypt** de la contraseña, entre comillas simples |
| `JWT_SECRET` | Clave para firmar los JWT |
| `ALLOWED_ORIGINS` | Orígenes del frontend separados por coma. Vacío = cualquier origen (solo dev) |
| `PORT` | Puerto de escucha. Render lo inyecta solo |

Para generar el hash bcrypt de la contraseña:

```bash
htpasswd -bnBC 10 "" TU_CONTRASENA | tr -d ':\n'
```

## Licencia

Proyecto educativo - Escuela Simón Bolívar 4-084
