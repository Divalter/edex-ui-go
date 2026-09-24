/*
 * Themed <select> pickers.
 *
 * Chromium (Electron) drew the option list of a <select> with the colors of
 * the element, so the original settings editor followed the theme. WebKitGTK
 * opens a native GTK menu instead, which ignores CSS. Every <select> is kept
 * in the DOM (hidden) so the original code keeps reading .value, and a button
 * plus an HTML option list styled like the rest of the UI stand in for it.
 */

let openList = null;

function closeList() {
    if (!openList) return;
    openList.list.remove();
    openList.button.classList.remove("open");
    openList = null;
}

function label(select) {
    const opt = select.options[select.selectedIndex];
    return opt ? opt.textContent : "";
}

function choose(select, button, index) {
    if (index >= 0 && index !== select.selectedIndex) {
        select.selectedIndex = index;
        select.dispatchEvent(new Event("input", {bubbles: true}));
        select.dispatchEvent(new Event("change", {bubbles: true}));
    }
    button.textContent = label(select);
}

function highlight(list, index) {
    const items = list.children;
    for (let i = 0; i < items.length; i++) items[i].classList.toggle("active", i === index);
    if (items[index]) items[index].scrollIntoView({block: "nearest"});
}

function open(select, button) {
    closeList();
    if (select.options.length === 0) return;

    const list = document.createElement("ul");
    list.className = "edex_select_list";
    Array.from(select.options).forEach((opt, i) => {
        const item = document.createElement("li");
        item.textContent = opt.textContent;
        if (i === select.selectedIndex) item.classList.add("selected");
        item.addEventListener("mousedown", e => e.preventDefault());
        item.addEventListener("click", () => {
            choose(select, button, i);
            closeList();
            button.focus();
        });
        item.addEventListener("mouseenter", () => {
            openList.index = i;
            highlight(list, i);
        });
        list.appendChild(item);
    });

    // Fixed on <body>: inside the modal, the augmented-ui clip-path would cut
    // the list at the modal edges.
    const modal = select.closest(".modal_popup");
    const z = modal ? (parseInt(getComputedStyle(modal).zIndex, 10) || 2500) + 1 : 3000;
    const rect = button.getBoundingClientRect();
    list.style.zIndex = z;
    list.style.left = `${rect.left}px`;
    list.style.minWidth = `${rect.width}px`;
    document.body.appendChild(list);

    const below = window.innerHeight - rect.bottom;
    if (list.offsetHeight > below && rect.top > below) {
        list.style.bottom = `${window.innerHeight - rect.top}px`;
        list.style.maxHeight = `${rect.top - 8}px`;
    } else {
        list.style.top = `${rect.bottom}px`;
        list.style.maxHeight = `${below - 8}px`;
    }

    button.classList.add("open");
    openList = {select, button, list, index: select.selectedIndex};
    highlight(list, openList.index);
}

function onKeydown(e, select, button) {
    const isOpen = openList && openList.select === select;
    const count = select.options.length;
    switch (e.key) {
        case "ArrowDown":
        case "ArrowUp": {
            const step = e.key === "ArrowDown" ? 1 : -1;
            if (isOpen) {
                openList.index = Math.min(count - 1, Math.max(0, openList.index + step));
                highlight(openList.list, openList.index);
            } else if (e.altKey) {
                open(select, button);
            } else {
                choose(select, button, Math.min(count - 1, Math.max(0, select.selectedIndex + step)));
            }
            break;
        }
        case "Enter":
        case " ":
            if (isOpen) {
                choose(select, button, openList.index);
                closeList();
            } else {
                open(select, button);
            }
            break;
        case "Escape":
            if (!isOpen) return;
            closeList();
            break;
        case "Tab":
            closeList();
            return;
        default:
            return;
    }
    e.preventDefault();
    e.stopPropagation();
}

function enhance(select) {
    if (select.dataset.edexSelect || select.multiple) return;
    select.dataset.edexSelect = "1";

    const button = document.createElement("button");
    button.type = "button";
    button.className = "edex_select";
    if (select.id) button.id = `${select.id}-picker`;
    button.textContent = label(select);
    button.addEventListener("click", () => {
        if (openList && openList.select === select) closeList();
        else open(select, button);
    });
    button.addEventListener("keydown", e => onKeydown(e, select, button));
    button.addEventListener("blur", () => {
        if (openList && openList.select === select) closeList();
    });
    // Keeps the label right when the original code changes the value.
    select.addEventListener("change", () => { button.textContent = label(select); });

    select.classList.add("edex_select_native");
    select.after(button);
}

function scan(root) {
    if (root.nodeType !== Node.ELEMENT_NODE) return;
    if (root.tagName === "SELECT") enhance(root);
    else root.querySelectorAll("select").forEach(enhance);
}

export function initSelects() {
    scan(document.body);
    new MutationObserver(mutations => {
        for (const m of mutations) {
            m.addedNodes.forEach(scan);
            // The picker's <select> went away with its modal.
            if (openList && !openList.select.isConnected) closeList();
        }
    }).observe(document.body, {childList: true, subtree: true});

    document.addEventListener("mousedown", e => {
        if (openList && !openList.list.contains(e.target) && e.target !== openList.button) closeList();
    }, true);
    window.addEventListener("resize", closeList);
    document.addEventListener("scroll", e => {
        if (openList && !openList.list.contains(e.target)) closeList();
    }, true);
}
