function fz {
  ( nohup fuzzy-clone cache update >/dev/null 2>&1 & )
  
  choice=$(fuzzy-clone)
  
  if [[ $? -ne 0 ]]; then
    return $?
  fi
  
  cd "$(echo "$choice" | tail --lines 1)"
}
