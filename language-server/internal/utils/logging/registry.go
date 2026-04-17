package logging

import (
	"io"
	"log"
	"strings"
	"sync"
)

// Module represents a loggable component in the system.
type Module string

const (
	ModuleLangServer Module = "langserver"
	ModuleHandlers   Module = "handlers"
	ModuleEventBus   Module = "eventbus"
	ModuleModules    Module = "modules"
	ModuleCRDs       Module = "crds"
	ModuleFilesystem Module = "filesystem"
	ModuleDocStore   Module = "docstore"
	ModulePerf       Module = "perf"
)

// AllModules lists all available modules for logging.
var AllModules = []Module{
	ModuleLangServer,
	ModuleHandlers,
	ModuleEventBus,
	ModuleModules,
	ModuleCRDs,
	ModuleFilesystem,
	ModuleDocStore,
	ModulePerf,
}

// registry manages module logging configuration.
type registry struct {
	mu             sync.RWMutex
	enabledModules map[Module]bool
	output         io.Writer
	loggers        map[Module]*log.Logger
}

var globalRegistry = &registry{
	enabledModules: make(map[Module]bool),
	loggers:        make(map[Module]*log.Logger),
}

// Init initializes the logging registry with the given output and enabled modules.
// This should be called once at startup before any components are created.
func Init(output io.Writer, enabledModules []Module) {
	globalRegistry.mu.Lock()
	defer globalRegistry.mu.Unlock()

	globalRegistry.output = output
	globalRegistry.enabledModules = make(map[Module]bool)
	globalRegistry.loggers = make(map[Module]*log.Logger)

	for _, m := range enabledModules {
		globalRegistry.enabledModules[m] = true
	}

	// Update PerfLogger based on whether perf module is enabled
	if globalRegistry.enabledModules[ModulePerf] && output != nil {
		PerfLogger = NewLogger(output)
	} else {
		PerfLogger = NopLogger()
	}
}

// LoggerFor returns a logger for the specified module.
// If the module is enabled, it returns a real logger; otherwise, it returns a NopLogger.
func LoggerFor(module Module) *log.Logger {
	globalRegistry.mu.RLock()
	if logger, ok := globalRegistry.loggers[module]; ok {
		globalRegistry.mu.RUnlock()
		return logger
	}
	globalRegistry.mu.RUnlock()

	globalRegistry.mu.Lock()
	defer globalRegistry.mu.Unlock()

	// Double-check after acquiring write lock
	if logger, ok := globalRegistry.loggers[module]; ok {
		return logger
	}

	var logger *log.Logger
	if globalRegistry.enabledModules[module] && globalRegistry.output != nil {
		logger = log.New(globalRegistry.output, "["+string(module)+"] ", log.LstdFlags|log.Lshortfile)
	} else {
		logger = NopLogger()
	}

	globalRegistry.loggers[module] = logger
	return logger
}

// ParseModules parses a comma-separated list of module names.
// The special value "all" enables all modules.
func ParseModules(input string) ([]Module, error) {
	if input == "" {
		return nil, nil
	}

	if input == "all" {
		return AllModules, nil
	}

	parts := strings.Split(input, ",")
	modules := make([]Module, 0, len(parts))
	validModules := make(map[Module]bool)
	for _, m := range AllModules {
		validModules[m] = true
	}

	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		m := Module(p)
		if !validModules[m] {
			// Return list of valid modules in error
			validNames := make([]string, len(AllModules))
			for i, vm := range AllModules {
				validNames[i] = string(vm)
			}
			return nil, &InvalidModuleError{
				Module:       p,
				ValidModules: validNames,
			}
		}
		modules = append(modules, m)
	}

	return modules, nil
}

// InvalidModuleError is returned when an invalid module name is specified.
type InvalidModuleError struct {
	Module       string
	ValidModules []string
}

func (e *InvalidModuleError) Error() string {
	return "invalid log module: " + e.Module + " (valid modules: " + strings.Join(e.ValidModules, ", ") + ")"
}
