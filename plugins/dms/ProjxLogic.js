.pragma library

function shortPath(p, homeDir) {
    if (homeDir !== "" && p.indexOf(homeDir) === 0)
        return "~" + p.substring(homeDir.length);
    return p;
}

function hasTag(project, tag) {
    var parts = (project.tags || "").split(",");
    var needle = (tag || "").toLowerCase();
    for (var i = 0; i < parts.length; i++) {
        if (parts[i].trim().toLowerCase() === needle)
            return true;
    }
    return false;
}

function parseProjects(raw) {
    var lines = (raw || "").split("\n");
    var projects = [];
    for (var i = 0; i < lines.length; i++) {
        var line = lines[i];
        if (line.trim() === "")
            continue;
        var f = line.split("\x1f");
        if (f.length < 4)
            continue;
        projects.push({
            id: f[0] || "",
            name: f[1] || "",
            path: f[2] || "",
            editor: f[3] || "",
            tags: f[4] || "",
            group: f[5] || "",
            favorite: f[6] === "1",
            gitBranch: f[7] || "",
            gitDirty: parseInt(f[8]) || 0,
            remote: f[9] || ""
        });
    }
    return projects;
}

function utilItems(filter) {
    var q = (filter || "").toLowerCase();
    var all = [
        { name: "Scan directories for projects", comment: "run pjx scan on configured roots", action: "util:scan", icon: "material:search", categories: ["projx"] },
        { name: "Import projects from Zed", comment: "one-time migration", action: "util:import-zed", icon: "material:download", categories: ["projx"] },
        { name: "Import from all editors", comment: "zed, vscode, cursor, ...", action: "util:import-all", icon: "material:download", categories: ["projx"] },
        { name: "Set default editor: zed", comment: "pjx config set default_editor zed", action: "util:editor-zed", icon: "material:code", categories: ["projx"] },
        { name: "Set default editor: vscode", comment: "pjx config set default_editor code", action: "util:editor-code", icon: "material:code", categories: ["projx"] },
        { name: "Set default editor: nvim", comment: "pjx config set default_editor nvim", action: "util:editor-nvim", icon: "material:code", categories: ["projx"] },
        { name: "Refresh project list", comment: "clear cache and reload", action: "util:refresh", icon: "material:refresh", categories: ["projx"] },
        { name: "List VS Code profiles", comment: "show known VS Code profiles", action: "util:profiles", icon: "material:person", categories: ["projx"] }
    ];
    var out = [];
    for (var i = 0; i < all.length; i++) {
        var it = all[i];
        if (q === "" || it.name.toLowerCase().indexOf(q) >= 0 || it.comment.toLowerCase().indexOf(q) >= 0)
            out.push(it);
    }
    return out;
}

function projectComment(p, homeDir) {
    var comment = shortPath(p.path || "", homeDir);
    if (p.gitBranch) {
        comment += "  [" + p.gitBranch;
        if (p.gitDirty > 0)
            comment += " +" + p.gitDirty;
        comment += "]";
    }
    if (p.tags)
        comment += "  " + p.tags;
    return comment;
}

function buildItems(projects, query, showProjects, maxResults, homeDir, loadError) {
    var items = [];
    var trimmed = query ? query.trim() : "";
    var lowerQuery = trimmed.toLowerCase();

    var tagFilter = null;
    var textFilter = trimmed;
    if (trimmed.indexOf("#") === 0) {
        tagFilter = trimmed.substring(1);
        textFilter = "";
        lowerQuery = "";
    }

    if (showProjects) {
        for (var i = 0; i < projects.length; i++) {
            var p = projects[i];
            if (tagFilter && !hasTag(p, tagFilter))
                continue;

            var pname = p.name || "";
            var ppath = p.path || "";
            if (textFilter === "" || pname.toLowerCase().indexOf(lowerQuery) >= 0 ||
                ppath.toLowerCase().indexOf(lowerQuery) >= 0) {
                items.push({
                    name: pname,
                    comment: projectComment(p, homeDir),
                    action: "project:" + ppath,
                    icon: p.favorite ? "material:star" : "material:folder",
                    categories: ["projx"]
                });
            }
            if (items.length >= maxResults)
                break;
        }
    }

    // Add project from path
    if (trimmed !== "" && (trimmed.indexOf("/") >= 0 || trimmed.indexOf("~") === 0 || trimmed.indexOf(".") === 0)) {
        var expanded = trimmed;
        if (expanded.indexOf("~/") === 0)
            expanded = homeDir + expanded.substring(1);
        var already = false;
        for (var j = 0; j < projects.length; j++) {
            if (projects[j].path === expanded) { already = true; break; }
        }
        if (!already) {
            items.push({
                name: "Add project: " + (expanded.split("/").pop() || expanded),
                comment: "create project from " + shortPath(expanded, homeDir),
                action: "add:" + expanded,
                icon: "material:add",
                categories: ["projx"]
            });
        }
    }

    // Profile create (searchable)
    if (lowerQuery.indexOf("profile") >= 0 || lowerQuery.indexOf("prof") >= 0) {
        var m = trimmed.match(/(?:create|new)\s+(.+)/i);
        var cpname = m ? m[1].trim() : "";
        if (cpname !== "") {
            items.push({ name: "Create VS Code profile: " + cpname, comment: "pjx profiles create " + cpname, action: "vscprofile:create:" + cpname, icon: "material:add", categories: ["projx"] });
        } else {
            items.push({ name: "Create VS Code profile", comment: "type: profile create <name>", action: "vscprofile:createprompt", icon: "material:add", categories: ["projx"] });
        }
    }

    // Utility entries (searchable by title/subtitle)
    var utils = utilItems(textFilter);
    for (var u = 0; u < utils.length; u++)
        items.push(utils[u]);

    if (loadError !== "" && items.length === 0) {
        items.push({ name: "pjx error", comment: loadError, action: "", icon: "material:error", categories: ["projx"] });
        return items;
    }
    if (items.length === 0) {
        items.push({ name: "No projects found", comment: "run `pjx scan` to discover projects", action: "", icon: "material:folder_off", categories: ["projx"] });
    }
    return items;
}
