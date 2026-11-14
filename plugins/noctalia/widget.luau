--!nonstrict

-- Bar widget showing the current project name (based on $PWD).

local cachedProjects = nil

local function trim(s)
	return (s or ""):match("^%s*(.-)%s*$")
end

local function splitFields(line)
	local fields = {}
	local start = 1
	local sep = "\31"
	while true do
		local sepStart, sepEnd = line:find(sep, start, true)
		if not sepStart then
			table.insert(fields, line:sub(start))
			break
		end
		table.insert(fields, line:sub(start, sepStart - 1))
		start = sepEnd + 1
	end
	return fields
end

local function loadProjects(onDone)
	noctalia.runAsync("pjx list --format=delimited 2>/dev/null", function(result)
		local projects = {}
		if result.exitCode == 0 and type(result.stdout) == "string" then
			for line in result.stdout:gmatch("[^\n]+") do
				local f = splitFields(line)
				if trim(f[1]) ~= "" then
					table.insert(projects, {
						name = trim(f[2]),
						path = trim(f[3]),
					})
				end
			end
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
	-- find longest matching project path prefix
	local best = nil
	local bestLen = 0
	for _, p in ipairs(projects) do
		if p.path ~= "" and pwd:sub(1, #p.path) == p.path and #p.path > bestLen then
			best = p.name
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
			barWidget.setText(name or "no project")
		end)
		return
	end
	local name = currentProjectName(cachedProjects)
	barWidget.setGlyph("folder")
	barWidget.setText(name or "no project")
end

function onClick()
	noctalia.runAsync("pjx tui >/dev/null 2>&1 &")
end
