(() => {
    /**
     * Where the user preference is saved locally.
     * @type {string}
     */
    const STORAGE_KEY = 'summit-theme';

    /**
     * I suppose the targer audience would probably prefer a dark mode to be default sadly.
     * @type {string}
     */
    const DEFAULT_THEME = 'dark';

    /**
     * Applies the theme and updates the local storage.
     * @param theme The theme to apply.
     */
    /**
     * Applies the theme and updates the local storage.
     * @param theme The theme to apply.
     */
    function applyTheme(theme) {
        const swatches = document.querySelectorAll('.theme-swatch');
        const themes = Array.from(swatches).map(btn => btn.dataset.theme);

        if (themes.length > 0 && !themes.includes(theme)) theme = DEFAULT_THEME;

        document.documentElement.setAttribute('data-theme', theme);
        localStorage.setItem(STORAGE_KEY, theme);
        document.cookie = `summit-theme=${theme};path=/;max-age=${60 * 60 * 24 * 365}`;

        swatches.forEach(btn => {
            btn.classList.toggle('active', btn.dataset.theme === theme);
        });

        /**
         * Gets the saved theme from local storage.
         * @returns {string|string} The saved theme or the default theme.
         */
        function getSavedTheme() {
            return localStorage.getItem(STORAGE_KEY) || DEFAULT_THEME;
        }

        /**
         * Initialises the theme switcher.
         */
        function init() {
            applyTheme(getSavedTheme());
            document.querySelectorAll('.theme-swatch').forEach(btn => {
                btn.addEventListener('click', () => applyTheme(btn.dataset.theme));
            });
        }

        if (document.readyState === 'loading') {
            document.addEventListener('DOMContentLoaded', init);
        } else {
            init();
        }
    }

)
    ();