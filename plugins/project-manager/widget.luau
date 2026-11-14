--!nonstrict

-- Bar widget showing the current project name (based on $PWD).

local common = require("./lib/common.luau")

local cachedProjects = nil

local function loadProjects(onDone)
    noctalia.runAsync("pjx list --format=delimited 2>/dev/null", function(result)
        local projects = {}
        if result.exitCode == 0 and type(result.stdout) == "string" then
            projects = common.parseProjects(result.stdout)
        end
        cachedProjects = projects
        onDone(projects)
    end)
end

local function currentProjectName(projects)
    local pwd = noctalia.getenv("PWD") or ""
    if pwd == "" then
        return nil
    end
    local best = nil
    local bestLen = 0
    for _, p in ipairs(projects) do
        if p.path ~= "" and pwd:sub(1, #p.path) == p.path and #p.path > bestLen then
            best = common.projectName(p)
            bestLen = #p.path
        end
    end
    return best
end

function update()
    noctalia.setUpdateInterval(5000)
    if not cachedProjects then
        loadProjects(function(projects)
            local name = currentProjectName(projects)
            barWidget.setGlyph("folder")
            barWidget.setText(name or noctalia.tr("no-project") or "no project")
        end)
        return
    end
    local name = currentProjectName(cachedProjects)
    barWidget.setGlyph("folder")
    barWidget.setText(name or noctalia.tr("no-project") or "no project")
end

function onClick()
    cachedProjects = nil
    update()
    noctalia.runAsync("pjx tui >/dev/null 2>&1 &")
end
