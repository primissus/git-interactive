This repo is used to create a list of commands (alias "gint", format "command subcommand [options]")under a single command called git-interact. The list of commands that this repo will enable are basically one:

  * branch (alias br):  will show a tabular list the list of branches in interactive mode. Shows branch name, last commit, last author, relative date. This command will enable navigation between branches with arrow
  functions. It should mimic most of the relevant interactions that git is capable of: sort (last commit, created, author, off), filter (Author, date (last 1 day, 3 days, 1 week, 1 month, year to date, 1 year), what else?),
  search (fuzzy find), . When a user presses "Enter" a menu will be displayed: Checkout, Delete (ask for confirmation, with an option to force delete that asks to enter word "force" as confirmation), pull, push, rename.
  Also in the list there should be an option above of the list (default focus on first branch) to create a new branch. Search is enabled with "/", "shift + N" (new) creates, "shift + C" (checkout), "shift + D" (delete),
  "shift + R" (rename), "p" pull, "shift + P" push (shortcuts ask for confirmation too). Command accepts arguments by default --sort | -S, --no-interactive | -I (just prints tabular), --new | -b create branch, --delete | -D
  delete, -i | --interactive, -m | --rename, --full | -F shows full command message line and date and author name, --short | -s only branch name. Tab is paginated taken height of the terminal/viewport (as less). Also supports VIM motions h j k l. "gint branch [branchname]" (no options) will show you a menu expecting
  input, supports fuzzy match (pull minus, Push mayus for disambiguate). Also add option to "copy sha", with shortcut. Colors show current and a Dot marker. Also supports select mode "shift + X" that shows bulk operations: archive (add tag `archive/<branch-name>` and delete branch), delete (confirmation asks to enter "delete all"), force delete (confirmation asks "force delete"). Also consider merge operation

  * worktree (alias wt): also a list, shows the worktree path (shortest: relative or absolute with "~" alias) , branch, commit and relative date. Also navigatable, operations: checkout (move to dir), search fuzzy, sort,
  filter, search, delete, fetch (or pull), push, rename branch, what else?. Same shortcuts ans arguments as branch.

  * graph-branch (alias grb): graphs branches with last commit only. options: --A | --not-all (current branch is base, all by default). Interactions similar to branch, accordingly.

  * log (alias lg): Similar to branches: shows commit sha (short), message (shortens), relative date, author (Short name, last name initial), and branches (local only by default, separated by commands) and worktree dir. Operations: similar as branch and cherry-pick "c" (confirmations shows no, yes, no commit, relevant options), selection supports cherry-pick too, squash, reset (confirmation no, soft, mixed, hard asks "reset hard"), merge. 

  * graph (alias gr): graphs commits. also -A | --not-all. 

  * rebase (alias reb): similar as interactive rebase in operations but pretty direct, at the end is submit, that asks confirmation.

  * merge (mrg): branches, confirmations shows no, yes, fast forward, etc.; what other options?

  * status (st): shows reference: @~/src/env/src/common/git/git-st.py. Interactive options: Enter to add command, if on conflicts can continue/abort, what else?

  * add: Ops: stash, unstash, restore, delete (clean), stash all, unstash all, what else?

Extra to consider TBD

  * remote (rem)

  * branch-remotes (brr)

## Architecture

stack is: golang, bubble tea library (tui), any extract args, a validation library (vyper on my mind, but you can search and peak a better if any), what other libraries?. Create a makefile, user architecture from /setup-context-architecture