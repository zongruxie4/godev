## 📌 Hoja de Ruta

### ✅ MVP (Versión Mínima Viable)  
### Frontend
- [x] Unificación y minificación de archivos CSS y JavaScript 
- [ ] cargar assets del directorio `web/theme` primero (assets handler)
- [ ] Generación automática de `web/public/index.html` si este no existe  
- [ ] Compilar iconos svg módulos en sprite único en `web/public/icons.svg`

### Servidor de Desarrollo
- [ ] Servidor de desarrollo integrado para servir archivos estáticos en `web/public`
- [ ] https integrado en desarrollo local
- [x] cerrar navegador al cerrar aplicación 
- [x] Ejecución navegador Chrome (tecla `w`)  
- [x] cambiar el tamaño de la ventana del navegador desde la tui

### Hot Reload
- [x] Detección de cambios en archivos HTML, CSS, y JS  
- [x] detección de cambios en archivos GO frontend para webAssembly y servidor backend
- [ ] detectar cambios en archivos SVG
- [ ] Recarga en caliente del navegador (Hot Reload)

### Backend
- [x] Detección de cambios en archivos del servidor  
- [ ] Reinicio automático si hay modificaciones  

### Configuración
- [x] Interfaz TUI minimalista para VS Code  
- [x] **Detección automática por estructura de directorios** ✅
- [x] **Eliminación completa de archivos de configuración** ✅
- [ ] Finalizar especificación de interacción TUI
- [ ] Agregar .gitignore automático


### 🚀 Mejoras Futuras  
- [ ] **Completar especificación TUI** para interacción final
- [ ] Modo producción: Artefactos optimizados y deploy
- [ ] Compatibilidad con servidores VPS
- [ ] Compatibilidad con Docker  
- [ ] Integrar ayudante IA


**Instalador web automático** que detectará e instalará todo automáticamente:
- [ ] **Detección automática** de dependencias instaladas
- [ ] **Instalación automática** de faltantes: Go, Git, TinyGo, Docker, GitHub CLI  
- [ ] **Setup completo** con un solo comando
- [ ] **Sin conocimiento técnico** requerido