# Storage boundary

File and object storage implementations belong here as features need them.
Keep storage independent of `site`, `view` and `model`; domain services receive
the storage capabilities they require through their constructors.

Browser assets belong in `view/assets`, not here. No persistent storage is
required by the initial application.
