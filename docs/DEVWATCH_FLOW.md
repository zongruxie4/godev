# TinyWasm DevWatch Flow

This diagram illustrates the internal processing of `tinywasm/devwatch` and its interactions with other system components (`client`, `server`, `assetmin`, `devbrowser`) as configured in `section-build.go`.

```mermaid
flowchart TD
    subgraph Initialization [Initialization Phase]
        New([New]) --> InitStruct[Init DevWatch & DepFinder]
        Start([FileWatcherStart]) --> LaunchLoop[Start watchEvents Goroutine]
        LaunchLoop --> InitReg[Initial Registration]
        
        InitReg --> WalkFS[Walk AppRootDir]
        WalkFS --> CheckType{File or Dir?}
        
        CheckType -- Dir --> AddWatch[Add to fsnotify Watcher]
        CheckType -- File --> SimEvent[Simulate 'Create' Event]
        SimEvent --> ProcInit["Process File (handlers)"]
    end

    subgraph Event_Loop [Event Processing Loop]
        direction TB
        EvIn(fsnotify Event) --> FilterUser{Is Ignored?}
        FilterUser -- Yes --> Drop1((Drop))
        FilterUser -- No --> Debounce{Smart Debounce}
        
        Debounce -- "Duplicate (Time & Hash)" --> Drop2((Drop))
        Debounce -- "New / Changed" --> Dispatch{Event Type}

        Dispatch -- "Dir Create" --> HandleDir[Handle Directory]
        HandleDir --> WatchNew[Add New Dir to Watcher]
        HandleDir --> NotifyFolder[Notify FolderEvents]

        Dispatch -- "File Event" --> HandleFile[Handle File]
        
        subgraph File_Handling [File Handling Logic]
            HandleFile --> IterHandlers[Iterate Handlers]
            IterHandlers --> MatchExt{Ext Match?}
            
            MatchExt -- Yes --> CheckGo{Is .go?}
            CheckGo -- Yes --> CheckDep{DepFinder: Is Mine?}
            CheckDep -- No --> NextHandler
            CheckDep -- Yes --> ExecHandler
            
            CheckGo -- No --> ExecHandler["Call handler.NewFileEvent()"]
            ExecHandler --> Result{Success?}
            Result -- Yes --> MarkReload[Mark for Reload]
            Result -- No --> NextHandler
        end
        
        MarkReload --> SchedReload[Schedule/Reschedule Reload]
    end

    subgraph Output [Output Actions]
        SchedReload -- "Timer Fires" --> TrigReload[Trigger BrowserReload]
        NotifyFolder -.-> ExtFolder[External Architecture Detection]
    end

    subgraph External_Components ["External Components (Interactions)"]
        direction LR
        
        subgraph Handlers [FilesEventHandlers Interface]
            Client[[tinywasm/client]]
            Server[[tinywasm/server]]
            AssetMin[[tinywasm/assetmin]]
        end

        Browser[[tinywasm/devbrowser]]
    end

    %% Connections to External Components
    ExecHandler -.-> Client
    ExecHandler -.-> Server
    ExecHandler -.-> AssetMin

    TrigReload -- "Invokes Callback" --> Browser
    Browser -- "Reloads Page" --> Chrome((Chrome))

    Initialization -.-> Event_Loop
```
