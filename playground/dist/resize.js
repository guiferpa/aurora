function resize() {
    const $divider = document.getElementById("divider");
    const $editor = document.getElementById("editor");
    const $actions = document.getElementById("actions");
    const $container = document.querySelector(".container");

    let isDragging = false;

    $divider.addEventListener("mousedown", () => {
        isDragging = true;
        document.body.style.cursor = "col-resize";
    });

    document.addEventListener("mousemove", (e) => {
        if (!isDragging) return;

        $editor.style.width = `${e.clientX}px`;
        $actions.style.width = `${$container.offsetWidth - e.clientX}px`;
    });

    document.addEventListener("mouseup", () => {
        isDragging = false;
        document.body.style.cursor = "default";
    });
}

document.addEventListener("DOMContentLoaded", resize);