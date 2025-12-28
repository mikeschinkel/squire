# Menu System Ideas — as of 2025-12-26

Main Menu
[^1] Explore [^2] Manage [^3] Compose — [^0] Main
[1] Commit code — [0] Help [9] Quit

Explore — Explore changes
[^1] Explore [^2] Manage [^3] Compose — [^0] Main
See [1] Status [2] Breaking [3] Other Changes [4] Tests — [0] Help [9] Quit

Expore Help
1. Status — Display output of `git status`
2. Breaking — Show list of known breaking changes (MAYBE using a bespoke UI)
3. Other Changes — Show list of maybe non-breaking changes (MAYBE using a bespoke UI)
4. Tests — Show info about tests (probably using a bespoke UI)

Manage — Manage file staging
[^1] Explore [^2] Manage [^3] Compose — [^0] Main
[1] Stage [2] Unstage [3] Group [4] Split  — [0] Help [9] Quit

Manage Help
1. Stage — Stage all files in the module OR just the files in a defined group, unstage all other files
2. Unstage — Unstage all files in the repo
3. Group — Generate a summary of cohesive groups of changes
4. Split — Defined which files get assigned to which groups of changes using a bespoke UI

Compose — Compose commit message
[^1] Explore [^2] Manage [^3] Compose — [^0] Main
[1] Staged [2] Generate [3] List [4] Merge [5] Edit — [0] Help [9] Quit

Compose Help
1. Staged — Show files that are staged
2. Generate — Generate a commit candidate for staged files
3. List — List current numbered commit canidates using a bespoke UI
4. Merge — Merge two commit canidates by number using a bespoke UI
5. Edit — Edit commit candidate using a bespoke UI


NOTES:
- Bespoke UI also certainly means a mini model UI using Bubble Tea
- Saving Staging State — I want to explore having Staging record the current staging status and write to a staging.json file in the project config directory. I don't want ot do this immediately, but I want to have it in mind so that we can have functionality like git stash but for staging so that Squire can save the staging state, allow the user to fiddle around with it, and then restore it in case they made an error. It should actually record this stage when it first loads so that it will always be there for a user to recover and not require them to have the forethought to save in advance. Further, it should be able to record multiple saves and be easy to manage, especially ones that are no longer relevant (unlike `git stash` where there are no tooling to manage old stashes.)
